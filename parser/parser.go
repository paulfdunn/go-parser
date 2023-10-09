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
	"regexp"
)

// Extract objects determine how extractions (Scanner.Extract) occur.
// The RegexString is converted to a regex and is run against the specified data column (after Split).
// Submatches is used to index submatches returned from regex.FindAllStringSubmatch(regex,-1) which are
// returned. The submatches are replaced with Token in the source data.
type Extract struct {
	Column      int
	RegexString string
	Submatch    int
	Token       string
	regex       *regexp.Regexp
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
// delimiter - Used by Split to split rows of data.
// extract - Extract objects; used for extracting values from rows into their own fields.
// negativeFilter - Regex used for negative filtering. Rows matching this value are excluded.
// positiveFilter - Regex used for positive filtering. Rows must match to be included.
// replace - Replacement values used for performing regex replacements on input data.
type Scanner struct {
	dataChan       chan string
	delimiter      *regexp.Regexp
	errorChan      chan error
	extract        []*Extract
	file           *os.File
	negativeFilter *regexp.Regexp
	positiveFilter *regexp.Regexp
	replace        []*Replacement
	scanner        *bufio.Scanner
}

// Extract takes an input row slice (call Split to split a row on scnr.delimiter)
// and applies the scnr.extract values to extract values from a column.
func (scnr *Scanner) Extract(row []string) []string {
	var extracts []string
	for _, extrct := range scnr.extract {
		if extrct.Column >= len(row) {
			continue
		}

		sbms := extrct.regex.FindAllStringSubmatch(row[extrct.Column], -1)
		for _, sbm := range sbms {
			if extrct.Submatch >= len(sbm) {
				continue
			}
			extracts = append(extracts, sbm[extrct.Submatch])
		}
		row[extrct.Column] = extrct.regex.ReplaceAllString(row[extrct.Column], extrct.Token)
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
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	scnr.OpenIoReaderScanner(file)
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
		for scnr.scanner.Scan() {
			row := scnr.scanner.Text()
			if err := scnr.scanner.Err(); err != nil {
				scnr.errorChan <- err
				continue
			}

			scnr.dataChan <- row
		}
		scnr.Shutdown()
		close(scnr.dataChan)
		close(scnr.errorChan)
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

// Shutdown performs an orderly shutdown on the scanner.
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
func Hash(input string) string {
	h := md5.New()
	io.WriteString(h, input)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// NewScanner is a constuctor for Scanners. See the Scanner definition for
// a description of inputs.
func NewScanner(negativeFilter string, positiveFilter string, delimiter string,
	replace []*Replacement, extract []*Extract) (*Scanner, error) {
	rgx, err := regexp.Compile(delimiter)
	if err != nil {
		return nil, err
	}
	scnr := &Scanner{delimiter: rgx}

	err = scnr.setFilter(false, negativeFilter)
	if err != nil {
		return nil, err
	}
	err = scnr.setFilter(true, positiveFilter)
	if err != nil {
		return nil, err
	}

	scnr.replace = make([]*Replacement, len(replace))
	for index := range replace {
		scnr.replace[index] = replace[index]
		rgx, err := regexp.Compile(replace[index].RegexString)
		if err != nil {
			return nil, err
		}
		scnr.replace[index].regex = rgx
	}

	scnr.extract = make([]*Extract, len(extract))
	for index := range extract {
		scnr.extract[index] = extract[index]
		rgx, err := regexp.Compile(extract[index].RegexString)
		if err != nil {
			return nil, err
		}
		scnr.extract[index].regex = rgx
	}

	return scnr, nil
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
