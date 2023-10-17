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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

// Inputs to constructor for scanner.
type Inputs struct {
	Delimiter          string
	ExpectedFieldCount int
	Extracts           []*Extract
	HashColumns        []int
	NegativeFilter     string
	PositiveFilter     string
	ProcessedDirectory string
	Replacements       []*Replacement
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
// delimiter - Regexp used by Split to split rows of data.
// extract - Extract objects; used for extracting values from rows into their own fields.
// negativeFilter - Regex used for negative filtering. Rows matching this value are excluded.
// positiveFilter - Regex used for positive filtering. Rows must match to be included.
// processedDirectory - When Read completes, move the file to this directory; empty string means the file is left in place.
// replace - Replacement values used for performing regex replacements on input data.
type Scanner struct {
	dataChan           chan string
	delimiter          *regexp.Regexp
	errorChan          chan error
	extract            []*Extract
	file               *os.File
	negativeFilter     *regexp.Regexp
	positiveFilter     *regexp.Regexp
	processedDirectory string
	replace            []*Replacement
	scanner            *bufio.Scanner
}

// Extract takes an input row slice (call Split to split a row on scnr.delimiter)
// and applies the scnr.extract values to extract values from a column.
func (scnr *Scanner) Extract(row []string) []string {
	var extracts []string
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
					continue
				}
				extracts = append(extracts, sbm[extrct.Submatch])
			}
			row[extrct.Columns[ec]] = extrct.regex.ReplaceAllString(row[extrct.Columns[ec]], extrct.Token)
		}
	}

	return extracts
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

		if scnr.processedDirectory != "" {
			err := os.Rename(processedFileName, filepath.Join(scnr.processedDirectory, filepath.Base(processedFileName)))
			if err != nil {
				scnr.errorChan <- err
			}
		}
	}()

	return scnr.dataChan, scnr.errorChan
}

// Replace applies the scnr.replace values to the supplied input row of data.
func (scnr *Scanner) Replace(row string) string {
	for _, rplc := range scnr.replace {
		row = rplc.regex.ReplaceAllString(row, rplc.Replacement)
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

// Split uses the scnr.delimiter to split the input data row.
func (scnr *Scanner) Split(row string) []string {
	return scnr.delimiter.Split(row, -1)
}

// Hash returns the hex string of the MD5 hash of the input. Call this on fields where
// values have been extracted in order to perform pareto analysis on the resulting hashes.
// This can also be used to reduce storage space when storing in a database by replacing
// multiple fields with a single hash, and keeping a separate table mapping hashes to
// original field values.
func Hash(input string) string {
	h := md5.New()
	io.WriteString(h, input)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// NewScanner is a constuctor for Scanners. See the Scanner definition for
// a description of inputs.
func NewScanner(inputs Inputs) (*Scanner, error) {
	rgx, err := regexp.Compile(inputs.Delimiter)
	if err != nil {
		return nil, err
	}
	scnr := &Scanner{delimiter: rgx}

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

	if _, err := os.Stat(inputs.ProcessedDirectory); inputs.ProcessedDirectory != "" && os.IsNotExist(err) {
		return nil, fmt.Errorf("processedDirectory does not exist, error: %+v", err)
	}
	scnr.processedDirectory = inputs.ProcessedDirectory

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
