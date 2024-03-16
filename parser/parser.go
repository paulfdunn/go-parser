// Author: Paul F. Dunn, https://github.com/paulfdunn/
// Original source location: https://github.com/paulfdunn/go-parser
// This code is licensed under the MIT license. Please keep this attribution when
// replicating/copying/reusing the code.
//
// Package parser was written to support parsing of log files that were written
// for human consumption and are generally difficult to parse. See the associtated
// test file for comments and examples with output.
package parser

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
)

// Extract objects determine how extractions (Scanner.Extract) occur.
// The RegexString is converted to a regex and is run against the specified data columns (after Split).
// Submatches is used to index submatches returned from regex.FindAllStringSubmatch(regex,-1) which are
// returned. The submatches are replaced with Token in the source data.
// Note on submatch indexing: The first item is the full match, so submatch indeces start at 1
// not zero (https://pkg.go.dev/regexp#Regexp.FindAllStringSubmatch)
type Extract struct {
	Columns     []int
	RegexString string
	Submatch    int
	Token       string
	regex       *regexp.Regexp
}

// Inputs to parser. This object is just used for unmarshalling inputs from a file.
// The values are then stored with the scanner; see Scanner for details.
type Inputs struct {
	DataDirectory           string
	ExpectedFieldCount      int
	Extracts                []*Extract
	HashColumns             []int
	InputDelimiter          string
	NegativeFilter          string
	OutputDelimiter         string
	PositiveFilter          string
	ProcessedInputDirectory string
	Replacements            []*Replacement
	SqlQuoteColumns         []int
}

// Replacement objects determine how replacements (Scanner.Replacement) occur.
// The RegexString is converted to a regex and is run against input row (unsplit),
// with matches being replaced by RegexString.
type Replacement struct {
	Replacement string
	RegexString string
	regex       *regexp.Regexp
}

// Scanner is the main object of this package.
// dataDirectory - Directory with input files.
// expectedFieldCount - Expected number of fields after calling Split.
// extract - Extract objects; used for extracting values from rows into their own fields.
// hashColumns - Column indeces (zero index) of Split data used to create the hash.
// inputDelimiter - Regexp used by Split to split rows of data.
// negativeFilter - Regex used for negative filtering. Rows matching this value are excluded.
// outDelimiter - String used to delimit parsed output data.
// positiveFilter - Regex used for positive filtering. Rows must match to be included.
// processedInputDirectory - When Read completes, move the file to this directory; empty string means the file is left in place.
// replace - Replacement values used for performing regex replacements on input data.
// sqlQuoteColumns - When using SQL ouput, these columns will be quoted.
type Scanner struct {
	HashColumns     []int
	HashCounts      map[string]int
	HashMap         map[string]string
	OutputDelimiter string

	dataChan                chan string
	dataDirectory           string
	errorChan               chan error
	expectedFieldCount      int
	extract                 []*Extract
	file                    *os.File
	inputDelimiter          *regexp.Regexp
	negativeFilter          *regexp.Regexp
	positiveFilter          *regexp.Regexp
	processedInputDirectory string
	replace                 []*Replacement
	scanner                 *bufio.Scanner
	sqlQuoteColumns         []int
}

// The hash can be output in a pure string format (I.E. "0xdeadbeef") or a format compatible
// for importing into Sqlite3 as a Blob (I.E. x'deadbeef').
type HashFormat int

const (
	HASH_FORMAT_STRING HashFormat = iota
	HASH_FORMAT_SQL
)

const (
	// Replacement regex that match this string will be replaced with unixmicro values to save
	// storage space.
	DATE_TIME_REGEX = "(\\d{4}-\\d{2}-\\d{2}[ -]\\d{2}:\\d{2}:\\d{2})"
)

// Extract takes an input row slice (call Split to split a row on scnr.inputDelimiter)
// and applies the scnr.extract values to extract values from a column.
func (scnr *Scanner) Extract(row []string) ([]string, []error) {
	var extracts []string
	errors := make([]error, 0)
	for _, extrct := range scnr.extract {
		// Allow empty Extracts that just have comments
		if extrct.RegexString == "" {
			continue
		}
		for ec := range extrct.Columns {
			if extrct.Columns[ec] >= len(row) {
				continue
			}

			sbms := extrct.regex.FindAllStringSubmatch(row[extrct.Columns[ec]], -1)
			for _, sbm := range sbms {
				if extrct.Submatch >= len(sbm) {
					errors = append(errors, fmt.Errorf("submatch index %d out of range for submatches:%+v, regex: %s",
						extrct.Submatch, sbm, extrct.RegexString))
					continue
				}
				extracts = append(extracts, sbm[extrct.Submatch])
			}
			row[extrct.Columns[ec]] = extrct.regex.ReplaceAllString(row[extrct.Columns[ec]], extrct.Token)
		}
	}

	return extracts, errors
}

// Filter takes in input row and applies the scnr.negativeFilter and
// scnr.positiveFilter. True means the row should be filtered (dropped),
// false means keep the row.
func (scnr *Scanner) Filter(row string) bool {
	if scnr.negativeFilter != nil && scnr.negativeFilter.MatchString(row) {
		return true
	}
	if scnr.positiveFilter != nil && !scnr.positiveFilter.MatchString(row) {
		return true
	}
	return false
}

// HashingEnabled is true when the inputs are specifying that hashing is to be performed; false otherwise.
func (scnr *Scanner) HashingEnabled() bool {
	if scnr.HashColumns != nil && len(scnr.HashColumns) > 0 {
		return true
	}
	return false
}

// OpenFileScanner convenience function to open a file based scanner.
func (scnr *Scanner) OpenFileScanner(filePath string) (err error) {
	scnr.file, err = os.Open(filePath)
	if err != nil {
		return err
	}

	scnr.OpenIoReaderScanner(scnr.file)
	return nil
}

// OpenIoReaderScanner opens a scanner using the supplied io.Reader. Callers reading
// from a file should call OpenFileScanner instead of this function.
func (scnr *Scanner) OpenIoReaderScanner(ior io.Reader) {
	scanner := bufio.NewScanner(ior)
	scnr.scanner = scanner
}

// Read starts a Go routine to read data from the input scanner and returns channels from
// which the caller can pull data and errors. Both data and error channels are buffered with
// buffer sizes databuffer and errorBuffer.
func (scnr *Scanner) Read(databuffer int, errorBuffer int) (<-chan string, <-chan error) {
	scnr.dataChan = make(chan string, databuffer)
	scnr.errorChan = make(chan error, errorBuffer)
	go func() {
		defer close(scnr.dataChan)
		defer close(scnr.errorChan)

		for scnr.scanner.Scan() {
			row := scnr.scanner.Text()
			if err := scnr.scanner.Err(); err != nil {
				scnr.errorChan <- err
				continue
			}

			scnr.dataChan <- row
		}

		// The name will not be available after Shutdown()
		processedFileName := scnr.file.Name()
		scnr.Shutdown()

		if scnr.processedInputDirectory != "" {
			err := os.Rename(processedFileName, filepath.Join(scnr.processedInputDirectory, filepath.Base(processedFileName)))
			if err != nil {
				scnr.errorChan <- err
			}
		}
	}()

	return scnr.dataChan, scnr.errorChan
}

// Replace applies the scnr.replace values to the supplied input row of data. The special case where
// RegexString == DATE_TIME_REGEX uses a function to replace a date time string with Unix epoch.
func (scnr *Scanner) Replace(row string) string {
	for _, rplc := range scnr.replace {
		if rplc.RegexString == DATE_TIME_REGEX {
			row = string(rplc.regex.ReplaceAllFunc([]byte(row), dateTimeToUnixEpoch))
		} else {
			row = rplc.regex.ReplaceAllString(row, rplc.Replacement)
		}
	}
	return row
}

// Shutdown performs an orderly shutdown on the scanner and is automatically called
// when Read completes. Callers should call shutdown if a scanner is created but not used.
func (scnr *Scanner) Shutdown() {
	if scnr.file != nil {
		scnr.file.Close()
	}
}

// Split uses the scnr.inputDelimiter to split the input data row. An error is returned if the
// resulting number of splits is not equal to Inputs.ExpectedFieldCount. But the data is
// returned and callers can choose to ignore the error if that is appropriate.
func (scnr *Scanner) Split(row string) ([]string, error) {
	splt := scnr.inputDelimiter.Split(row, -1)
	if len(splt) != scnr.expectedFieldCount {
		return splt, fmt.Errorf("Split expectedFieldCount: %d, actual: %d", scnr.expectedFieldCount, len(splt))
	}
	return splt, nil
}

// SplitsExcludeHashColumns creates a version of Split data that doesn't included the hash columns.
// It also calculates the hash of splits and adds the hash to hashMap and hashCount
func (scnr *Scanner) SplitsExcludeHashColumns(splits []string, hashFormat HashFormat) ([]string, error) {
	// Create the hash
	sortedHashColumns := sort.IntSlice(scnr.HashColumns)
	hashSplits := make([]string, 0, len(sortedHashColumns))
	for _, v := range sortedHashColumns {
		hashSplits = append(hashSplits, splits[v])
	}
	hashString := strings.Join(hashSplits, scnr.OutputDelimiter)
	hash, err := Hash(hashString, hashFormat)
	if err != nil {
		return nil, err
	}
	scnr.HashMap[hash] = hashString
	scnr.HashCounts[hash] += 1

	// Create a version of splits that doesn't included the hash columns.
	// The idea is to substitute multiple columns with the hash.
	// Make a copy of sortedHashColumns that is used to create a list of splits
	// that don't include hash columns.
	shc := make([]int, len(sortedHashColumns))
	copy(shc, sortedHashColumns)
	splitsExcludeHashColumns := make([]string, 0, len(splits)-len(sortedHashColumns)+1)
	hashInserted := false
	for i := range splits {
		if len(shc) > 0 {
			// Check each index in splits. If the index is in the slice of (sorted) hashed
			// columns, don't include the split in the return data.
			// The hash is inserted at the first hashed (dropped) column.
			if i == shc[0] {
				if !hashInserted {
					hashInserted = true
					splitsExcludeHashColumns = append(splitsExcludeHashColumns, hash)
				}
				shc = shc[1:]
				continue
			}
		}
		splitsExcludeHashColumns = append(splitsExcludeHashColumns, splits[i])
	}

	return splitsExcludeHashColumns, nil
}

// SplitsToSql will take a Split splits and convert it into an SQL INSERT INTO statement.
// All values are output as text. numColumns of Values will be provided, NULL padded.
// The table should be created with nullable text columns to receive as many extracts as
// might be produced. If the length of splits exceeds numColumns, the VALUES will be truncated.
// splits will be padded according to Scanner.SqlQuoteColumns, all extracts are quoted.
func (scnr *Scanner) SplitsToSql(numColumns int, table string, splits []string, extracts []string) string {
	out := fmt.Sprintf("INSERT OR IGNORE INTO %s VALUES(", table)
	sliceIn := append(splits, extracts...)

	// Turn splits and extract into a comma separated string, quoted as specified.
	outs := make([]string, 0, len(sliceIn))
	for i := 0; i < min(len(sliceIn), numColumns); i++ {
		if slices.Contains(scnr.sqlQuoteColumns, i) || i >= len(splits) {
			outs = append(outs, fmt.Sprintf("'%s'", sliceIn[i]))
		} else {
			outs = append(outs, sliceIn[i])
		}
	}
	out += strings.Join(outs, ",")

	padLength := numColumns - len(outs)
	if padLength > 0 {
		pad := make([]string, padLength)
		for i := 0; i < padLength; i++ {
			pad[i] = "NULL"
		}
		out += "," + strings.Join(pad, ",")
	}
	out += ");"
	return out
}

// Hash returns the hex string of the MD5 hash of the input. Call this on fields where
// values have been extracted in order to perform pareto analysis on the resulting hashes.
// This can also be used to reduce storage space when storing in a database by replacing
// multiple fields with a single hash, and keeping a separate table mapping hashes to
// original field values.
func Hash(input string, format HashFormat) (string, error) {
	h := md5.New()
	var out string
	_, err := io.WriteString(h, input)
	if err != nil {
		return "", err
	}
	switch format {
	case HASH_FORMAT_STRING:
		out = fmt.Sprintf("'0x%x'", h.Sum(nil))
	case HASH_FORMAT_SQL:
		out = fmt.Sprintf("x'%x'", h.Sum(nil))
	}
	return out, err
}

// Hash8 implements the djb2 hash described here: http://www.cse.yorku.ca/~oz/hash.html
// and returns only 8 bytes.
func Hash8(input string, format HashFormat) (string, error) {
	hash := 0
	for _, v := range input {
		hash = (hash * 33) + int(v)
	}
	if hash < 0 {
		hash = -hash
	}
	var out string
	switch format {
	case HASH_FORMAT_STRING:
		out = fmt.Sprintf("'0x%x'", hash)
	case HASH_FORMAT_SQL:
		out = fmt.Sprintf("x'%016x'", hash)
	}
	return out, nil
}

// NewInputs unmarshalls a JSON file into a new Inputs object.
func NewInputs(filePath string) (*Inputs, error) {
	inputBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	inputs := Inputs{}
	err = json.Unmarshal(inputBytes, &inputs)
	if err != nil {
		return nil, err
	}

	return &inputs, nil
}

// NewScanner is a constuctor for Scanners. See the Scanner definition for
// a description of inputs.
func NewScanner(inputs Inputs) (*Scanner, error) {
	hashMap := make(map[string]string)
	hashCounts := make(map[string]int)

	rgx, err := regexp.Compile(inputs.InputDelimiter)
	if err != nil {
		return nil, err
	}
	scnr := &Scanner{
		HashColumns:        inputs.HashColumns,
		HashCounts:         hashCounts,
		HashMap:            hashMap,
		OutputDelimiter:    inputs.OutputDelimiter,
		dataDirectory:      inputs.DataDirectory,
		inputDelimiter:     rgx,
		expectedFieldCount: inputs.ExpectedFieldCount,
		sqlQuoteColumns:    inputs.SqlQuoteColumns,
	}

	err = scnr.setFilter(false, inputs.NegativeFilter)
	if err != nil {
		return nil, err
	}
	err = scnr.setFilter(true, inputs.PositiveFilter)
	if err != nil {
		return nil, err
	}

	scnr.replace = make([]*Replacement, len(inputs.Replacements))
	for index := range inputs.Replacements {
		scnr.replace[index] = inputs.Replacements[index]
		rgx, err := regexp.Compile(inputs.Replacements[index].RegexString)
		if err != nil {
			return nil, err
		}
		scnr.replace[index].regex = rgx
	}

	scnr.extract = make([]*Extract, len(inputs.Extracts))
	for index := range inputs.Extracts {
		scnr.extract[index] = inputs.Extracts[index]
		rgx, err := regexp.Compile(inputs.Extracts[index].RegexString)
		if err != nil {
			return nil, err
		}
		scnr.extract[index].regex = rgx
	}

	if _, err := os.Stat(inputs.ProcessedInputDirectory); inputs.ProcessedInputDirectory != "" && os.IsNotExist(err) {
		return nil, fmt.Errorf("processedInputDirectory does not exist, error: %+v", err)
	}
	scnr.processedInputDirectory = inputs.ProcessedInputDirectory

	return scnr, nil
}

// Convenience function to sort a map of hashes based on counts. Used to help develop
// extracts and hashes in order to reduce the total number of hashes.
func SortedHashMapCounts(inputMap map[string]int) []string {
	hashes := make([]string, 0, len(inputMap))

	for hash := range inputMap {
		hashes = append(hashes, hash)
	}
	sort.SliceStable(hashes, func(i, j int) bool {
		return inputMap[hashes[i]] > inputMap[hashes[j]]
	})

	return hashes
}

// dateTimeToUnixEpoch is used to convert strings that match DATE_TIME_REGEX into Unix epoch
func dateTimeToUnixEpoch(input []byte) []byte {
	t, _ := time.Parse(time.DateTime, string(input))
	return []byte(fmt.Sprint(t.Unix()))
}

// setFilter is a convenience function to set the Scanner filters from inputs.
func (scnr *Scanner) setFilter(positive bool, regex string) error {
	if regex == "" {
		return nil
	}

	rgx, err := regexp.Compile(regex)
	if err != nil {
		return err
	}

	if positive {
		scnr.positiveFilter = rgx
	} else {
		scnr.negativeFilter = rgx
	}
	return nil
}
