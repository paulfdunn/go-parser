// Author: Paul F. Dunn, https://github.com/paulfdunn/
// Original source location: https://github.com/paulfdunn/go-parser
// This code is licensed under the MIT license. Please keep this attribution when
// replicating/copying/reusing the code.
package parser

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	wd, _             = os.Getwd()
	testDataDirectory = filepath.Join(wd, "./test")
)

// openFileScanner is a convenience test function for getting a NewScanner to the test file.
// Callers must call scnr.Shutdown() when not calling Read.
func openFileScanner(testDataFilePath string, inputs Inputs) *Scanner {
	scnr, err := NewScanner(inputs)
	if err != nil {
		var t *testing.T
		t.Errorf("calling OpenScanner: %s", err)
		return nil
	}
	scnr.OpenFileScanner(testDataFilePath)
	return scnr
}

// ExampleScanner_OpenFileScanner shows how to open a file for processing.
func ExampleScanner_OpenFileScanner() {
	defaultInputs, _ := NewInputs("./test/testInputs.json")
	scnr, err := NewScanner(*defaultInputs)
	if err != nil {
		var t *testing.T
		t.Errorf("calling OpenScanner: %s", err)
	}
	scnr.OpenFileScanner(filepath.Join(testDataDirectory, "test_read.txt"))
	defer scnr.Shutdown()

	//Output:
}

// ExampleScanner_OpenIoReaderScanner shows how to open an io.Reader for processing.
// Note that a file is used for convenience in calling OpenIoReaderScanner. When
// processing files, use the OpenFileScanner convenience function.
func ExampleScanner_OpenIoReaderScanner() {
	file, err := os.Open(filepath.Join(testDataDirectory, "test_read.txt"))
	if err != nil {
		var t *testing.T
		t.Errorf("calling os.Open: %s", err)
	}

	defaultInputs, _ := NewInputs("./test/testInputs.json")
	scnr, err := NewScanner(*defaultInputs)
	if err != nil {
		var t *testing.T
		t.Errorf("calling OpenIoReaderScanner: %s", err)
	}
	scnr.OpenIoReaderScanner(file)
	defer scnr.Shutdown()

	//Output:
}

// ExampleScanner_Read shows how to read data, with no other processing.
func ExampleScanner_Read() {
	defaultInputs, _ := NewInputs("./test/testInputs.json")
	scnr := openFileScanner(filepath.Join(testDataDirectory, "test_read.txt"), *defaultInputs)
	fmt.Println("Read all the test data")
	dataChan, errorChan := scnr.Read(100, 100)
	for row := range dataChan {
		fmt.Println(row)
	}
	for err := range errorChan {
		fmt.Println(err)
	}

	// Output:
	// Read all the test data
	// 2023-10-07 12:00:00.00 MDT  0         0         notification  debug          multi word type     sw_a          Debug SW message
	// 2023-10-07 12:00:00.01 MDT  1         001       notification  info           SingleWordType      sw_b          Info SW message
	// 2023-10-07 12:00:00.02 MDT  1         002       status        info           alphanumeric value  sw_a          Message with alphanumberic value abc123def
}

// ExampleScanner_Read_move shows how to read data and move the file when when processing is complete.
func TestScanner_Read_move(t *testing.T) {
	// Duplicate the existing test file in a temp dir so we can test moving the file on completion.
	testFileName := "test_read.txt"
	wd, _ := os.Getwd()
	testDirectory := filepath.Join(wd, "test")
	testFilePath := filepath.Join(testDirectory, testFileName)
	testFileBytes, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Errorf("calling os.ReadFile: %s", err)
	}
	tmpInputFilePath := filepath.Join(t.TempDir(), testFileName)
	err = os.WriteFile(tmpInputFilePath, testFileBytes, 0644)
	if err != nil {
		t.Errorf("calling os.WriteFile: %s", err)
	}

	defaultInputs, _ := NewInputs("./test/testInputs.json")
	scnr := openFileScanner(tmpInputFilePath, *defaultInputs)
	scnr.processedInputDirectory = t.TempDir()
	fmt.Println("Read all the test data")
	dataChan, errorChan := scnr.Read(100, 100)
	for row := range dataChan {
		fmt.Println(row)
	}
	for err := range errorChan {
		fmt.Println(err)
	}

	// Read the file from the processedInputDirectory and byte compare to the original
	testFileBytesAfterMove, err := os.ReadFile(filepath.Join(scnr.processedInputDirectory, testFileName))
	if err != nil {
		t.Errorf("calling os.ReadFile: %s", err)
	}
	if !bytes.Equal(testFileBytes, testFileBytesAfterMove) {
		t.Errorf("bytes not equal: \n%s\n%s", testFileBytes, testFileBytesAfterMove)
	}

	// Output:
	// Read all the test data
	// 2023-10-07 12:00:00.00 MDT  0         0         notification  debug          multi word type     sw_a          Debug SW message
	// 2023-10-07 12:00:00.01 MDT  1         001       notification  info           SingleWordType      sw_b          Info SW message
	// 2023-10-07 12:00:00.02 MDT  1         002       status        info           alphanumeric value  sw_a          Message with alphanumberic value abc123def
}

// ExampleScanner_Filter_negative shows how to use the negative filter to remove lines not matching a pattern.
// Note the comment line and line with 'negative filter' are not included in the output.
func ExampleScanner_Filter_negative() {
	// The '\s+' is used in the filter only to show that it is a regex; a space could have been used.
	defaultInputs, _ := NewInputs("./test/testInputs.json")
	defaultInputs.NegativeFilter = `#|negative\s+filter`
	scnr := openFileScanner(filepath.Join(testDataDirectory, "test_filter.txt"), *defaultInputs)
	dataChan, errorChan := scnr.Read(100, 100)
	fullData := []string{}
	filteredData := []string{}
	for row := range dataChan {
		fullData = append(fullData, row)
		if !scnr.Filter(row) {
			filteredData = append(filteredData, row)
		}
	}
	for err := range errorChan {
		fmt.Println(err)
	}

	fmt.Println("\nInput data:")
	fmt.Printf("%+v", strings.Join(fullData, "\n"))
	fmt.Println("\n\nFiltered data:")
	fmt.Printf("%+v", strings.Join(filteredData, "\n"))

	// Output:
	//
	// Input data:
	// # Comment line
	// 2023-10-07 12:00:00.00 MDT  0         0         notification  debug          will it filter     sw_a          Debug SW message
	// 2023-10-07 12:00:00.01 MDT  1         001       notification  info           negative filter      sw_b          Info SW message
	// 2023-10-07 12:00:00.02 MDT  1         002       status        info           will it filter  sw_a          Message with alphanumberic value abc123def
	//
	// Filtered data:
	// 2023-10-07 12:00:00.00 MDT  0         0         notification  debug          will it filter     sw_a          Debug SW message
	// 2023-10-07 12:00:00.02 MDT  1         002       status        info           will it filter  sw_a          Message with alphanumberic value abc123def
}

// ExampleScanner_Filter_positive shows how to use the positive filter to include lines matching a pattern.
// Note lines without a timestamp are not included in the output
func ExampleScanner_Filter_positive() {
	defaultInputs, _ := NewInputs("./test/testInputs.json")
	defaultInputs.PositiveFilter = `\d{4}-\d{2}-\d{2}[ -]\d{2}:\d{2}:\d{2}\.\d{2}\s+[a-zA-Z]{2,5}`
	scnr := openFileScanner(filepath.Join(testDataDirectory, "test_filter.txt"), *defaultInputs)
	dataChan, errorChan := scnr.Read(100, 100)
	fullData := []string{}
	filteredData := []string{}
	for row := range dataChan {
		fullData = append(fullData, row)
		if !scnr.Filter(row) {
			filteredData = append(filteredData, row)
		}
	}
	for err := range errorChan {
		fmt.Println(err)
	}

	fmt.Println("\nInput data:")
	fmt.Printf("%+v", strings.Join(fullData, "\n"))
	fmt.Println("\n\nFiltered data:")
	fmt.Printf("%+v", strings.Join(filteredData, "\n"))

	// Output:
	//
	// Input data:
	// # Comment line
	// 2023-10-07 12:00:00.00 MDT  0         0         notification  debug          will it filter     sw_a          Debug SW message
	// 2023-10-07 12:00:00.01 MDT  1         001       notification  info           negative filter      sw_b          Info SW message
	// 2023-10-07 12:00:00.02 MDT  1         002       status        info           will it filter  sw_a          Message with alphanumberic value abc123def
	//
	// Filtered data:
	// 2023-10-07 12:00:00.00 MDT  0         0         notification  debug          will it filter     sw_a          Debug SW message
	// 2023-10-07 12:00:00.01 MDT  1         001       notification  info           negative filter      sw_b          Info SW message
	// 2023-10-07 12:00:00.02 MDT  1         002       status        info           will it filter  sw_a          Message with alphanumberic value abc123def
}

// ExampleScanner_Replace shows how to use the Replace function to replace text that didn't
// include a delimiter with text that does have a delimiter. The delimiter in this example is two or more
// spaces. More than 2 consecutive spaces are also replaced with 2 spaces to enable splitting on a
// consistent delimiter.
func ExampleScanner_Replace() {
	delimiter := `\s\s`
	delimiterString := "  "
	// Note the order of the Replacements may be important. In this example a string that didn't include
	// delimiters is replaced with one that does. The next replacement is to replace more than 2
	// consecutive spaces with the delimiter, which is 2 consecutive spaces. If the order of the
	// Replacements is reveresed, there will be more than 2 spaces seperating the poorly delimited
	rplc := []*Replacement{
		{RegexString: "(class poor delimiting)", Replacement: delimiterString + "${1}" + delimiterString},
		{RegexString: `\s\s+`, Replacement: delimiterString},
	}
	defaultInputs, _ := NewInputs("./test/testInputs.json")
	defaultInputs.InputDelimiter = delimiter
	defaultInputs.Replacements = rplc
	scnr := openFileScanner(filepath.Join(testDataDirectory, "test_replace.txt"), *defaultInputs)
	dataChan, errorChan := scnr.Read(100, 100)
	fullData := []string{}
	replacedData := []string{}
	for row := range dataChan {
		fullData = append(fullData, row)
		row = scnr.Replace(row)
		replacedData = append(replacedData, row)
	}
	for err := range errorChan {
		fmt.Println(err)
	}

	fmt.Println("\nInput data:")
	fmt.Printf("%+v", strings.Join(fullData, "\n"))
	fmt.Println("\n\nReplaced data:")
	fmt.Printf("%+v", strings.Join(replacedData, "\n"))

	// Output:
	//
	// Input data:
	// 2023-10-07 12:00:00.00 MDT  0         000 class poor delimiting debug embedded values            sw_a          Message with embedded hex flag=0x01 and integer flag = 003
	//
	// Replaced data:
	// 2023-10-07 12:00:00.00 MDT  0  000  class poor delimiting  debug embedded values  sw_a  Message with embedded hex flag=0x01 and integer flag = 003
}

// ExampleScanner_Split shows how to use the Split function. In this case the data is then
// Join'ed back together just for output purposed.
// Note that the call to Split drops the error that ExpectedFieldCount was incorrect.
// callers can choose to enforce the error, or not.
func ExampleScanner_Split() {
	delimiter := `\s\s+`
	delimiterString := "  "
	defaultInputs, _ := NewInputs("./test/testInputs.json")
	defaultInputs.InputDelimiter = delimiter
	defaultInputs.ExpectedFieldCount = 8
	defaultInputs.Replacements = []*Replacement{{RegexString: `\s\s+`, Replacement: delimiterString}}
	scnr := openFileScanner(filepath.Join(testDataDirectory, "test_split.txt"), *defaultInputs)
	dataChan, errorChan := scnr.Read(100, 100)
	fullData := []string{}
	splitData := []string{}
	for row := range dataChan {
		fullData = append(fullData, row)
		splits, _ := scnr.Split(row)
		splitData = append(splitData, strings.Join(splits, "|"))
	}
	for err := range errorChan {
		fmt.Println(err)
	}

	fmt.Println("\nInput data:")
	fmt.Printf("%+v", strings.Join(fullData, "\n"))
	fmt.Println("\n\nSplit data:")
	fmt.Printf("%+v", strings.Join(splitData, "\n"))

	// Output:
	//
	// Input data:
	// 2023-10-07 12:00:00.00 MDT  0         0         notification  debug          multi word type     sw_a          Debug SW message
	// 2023-10-07 12:00:00.01 MDT  1         001       notification  info           SingleWordType      sw_b          Info SW message
	// 2023-10-07 12:00:00.02 MDT  1         002       status        info           alphanumeric value  sw_a          Message with alphanumberic value abc123def
	// 2023-10-07 12:00:00.03 MDT  1         003       status        info           alphanumeric value  sw_a          Message   with   extra   delimiters
	//
	// Split data:
	// 2023-10-07 12:00:00.00 MDT|0|0|notification|debug|multi word type|sw_a|Debug SW message
	// 2023-10-07 12:00:00.01 MDT|1|001|notification|info|SingleWordType|sw_b|Info SW message
	// 2023-10-07 12:00:00.02 MDT|1|002|status|info|alphanumeric value|sw_a|Message with alphanumberic value abc123def
	// 2023-10-07 12:00:00.03 MDT|1|003|status|info|alphanumeric value|sw_a|Message|with|extra|delimiters
}

// ExampleScanner_Extract shows how to extract data and hash a field.
// Note that the order of the extracts is based on the order of the extract expression evaluation, NOT
// the order of the data in the original string.
// Hash - Note that hashing a field after extracting unique data results in equal hashes. This is
// useful in order to calculate a pareto of message types regardless of some unique data.
func ExampleScanner_Extract_andHash() {
	delimiter := `\s\s+`
	delimiterString := "  "

	extracts := []*Extract{
		{
			// capture string that starts with alpha or number, and contains alpha, number, [_.-:], that has leading space delimited
			Columns:     []int{7},
			RegexString: "(^|\\s+)(([0-9]+[a-zA-Z_\\.-]|[a-zA-Z_\\.-]+[0-9])[a-zA-Z0-9\\.\\-_:]*)",
			Token:       "${1}{}",
			Submatch:    2,
		},
		{
			// capture word or [\\._] preceeded by' word='
			Columns:     []int{7},
			RegexString: "(^|\\s+)([\\w]+[:=])([\\w:\\._]+)",
			Token:       "${1}${2}{}",
			Submatch:    3,
		},
		{
			// capture word or [\\.] in paretheses
			Columns:     []int{7},
			RegexString: "(\\()([\\w:\\.]+)(\\))",
			Token:       "${1}{}${3}",
			Submatch:    2,
		},
		{
			// capture hex number preceeded by space
			Columns:     []int{7},
			RegexString: "(^|\\s+)(0x[a-fA-F0-9]+)",
			Token:       "${1}{}",
			Submatch:    2,
		},
		{
			// capture number and [\\.:_] preceeded by space
			Columns:     []int{7},
			RegexString: "(^|\\s+)([0-9\\.:_]+)",
			Token:       "${1}{}",
			Submatch:    2,
		},
	}
	defaultInputs, _ := NewInputs("./test/testInputs.json")
	defaultInputs.InputDelimiter = delimiter
	defaultInputs.Replacements = []*Replacement{{RegexString: `\s\s+`, Replacement: delimiterString}}
	defaultInputs.Extracts = extracts
	defaultInputs.HashColumns = []int{3, 4, 5, 7}
	scnr := openFileScanner(filepath.Join(testDataDirectory, "test_extract.txt"), *defaultInputs)
	dataChan, errorChan := scnr.Read(100, 100)
	fullData := []string{}
	extractData := []string{}
	extractExcludeColumnsData := []string{}
	for row := range dataChan {
		splits, _ := scnr.Split(row)
		fullData = append(fullData, strings.Join(splits, "|"))
		extracts, _ := scnr.Extract(splits)
		extractData = append(extractData, strings.Join(splits, "|")+
			"|EXTRACTS|"+strings.Join(extracts, "|")+
			"| hash:"+Hash(splits[3]+splits[4]+splits[5]+splits[7]))
		extractExcludeColumnsData = append(extractExcludeColumnsData, strings.Join(scnr.SplitsExcludeHashColumns(splits), "|")+
			"|EXTRACTS|"+strings.Join(extracts, "|")+
			"| hash:"+Hash(splits[3]+splits[4]+splits[5]+splits[7]))
	}
	for err := range errorChan {
		fmt.Println(err)
	}

	fmt.Printf("Hashing is enabled: %t", scnr.HashingEnabled())
	fmt.Println("\nInput data:")
	fmt.Printf("%+v", strings.Join(fullData, "\n"))
	fmt.Println("\n\nExtract(ed) data:")
	fmt.Printf("%+v", strings.Join(extractData, "\n"))
	fmt.Println("\n\nExtract(ed) data excluding hashed columns:")
	fmt.Printf("%+v", strings.Join(extractExcludeColumnsData, "\n"))

	// Output:
	// Hashing is enabled: true
	// Input data:
	// 2023-10-07 12:00:00.00 MDT|0|0|notification|debug|multi word type|sw_a|Unit 12.Ab.34 message (789)
	// 2023-10-07 12:00:00.01 MDT|1|001|notification|info|SingleWordType|sw_b|Info SW version = 1.2.34 release=a.1.1
	// 2023-10-07 12:00:00.02 MDT|1|002|status|info|alphanumeric value|sw_a|Message with alphanumberic value abc123def
	// 2023-10-07 12:00:00.03 MDT|1|003|status|info|alphanumeric value|sw_a|val:1 flag:x20 other:X30 on 127.0.0.1:8080
	// 2023-10-07 12:00:00.04 MDT|1|004|status|info|alphanumeric value|sw_a|val=2 flag = 30 other 3.cd on (ABC.123_45)
	// 2023-10-07 12:00:00.05 MDT|1|005|status|info|alphanumeric value|sw_a|val=3 flag = 40 other 4.ef on (DEF.678_90)
	// 2023-10-07 12:00:00.06 MDT|1|006|status|info|alphanumeric value|sw_a|val=4 flag = 50 other 5.gh on (GHI.098_76)
	//
	// Extract(ed) data:
	// 2023-10-07 12:00:00.00 MDT|0|0|notification|debug|multi word type|sw_a|Unit {} message ({})|EXTRACTS|12.Ab.34|789| hash:a5a3dba744d3c6f1372f888f54447553
	// 2023-10-07 12:00:00.01 MDT|1|001|notification|info|SingleWordType|sw_b|Info SW version = {} release={}|EXTRACTS|1.2.34|a.1.1| hash:9bd3989cf85b232ddadd73a1a312b249
	// 2023-10-07 12:00:00.02 MDT|1|002|status|info|alphanumeric value|sw_a|Message with alphanumberic value {}|EXTRACTS|abc123def| hash:7f0e8136c3aec6bbde74dfbad17aef1c
	// 2023-10-07 12:00:00.03 MDT|1|003|status|info|alphanumeric value|sw_a|val:{} flag:{} other:{} on {}|EXTRACTS|127.0.0.1:8080|1|x20|X30| hash:4907fb17a4212e2e09897fafa1cb758a
	// 2023-10-07 12:00:00.04 MDT|1|004|status|info|alphanumeric value|sw_a|val={} flag = {} other {} on ({})|EXTRACTS|3.cd|2|ABC.123_45|30| hash:1b7739c1e24d3a837e7821ecfb9a1be1
	// 2023-10-07 12:00:00.05 MDT|1|005|status|info|alphanumeric value|sw_a|val={} flag = {} other {} on ({})|EXTRACTS|4.ef|3|DEF.678_90|40| hash:1b7739c1e24d3a837e7821ecfb9a1be1
	// 2023-10-07 12:00:00.06 MDT|1|006|status|info|alphanumeric value|sw_a|val={} flag = {} other {} on ({})|EXTRACTS|5.gh|4|GHI.098_76|50| hash:1b7739c1e24d3a837e7821ecfb9a1be1
	//
	// Extract(ed) data excluding hashed columns:
	// 2023-10-07 12:00:00.00 MDT|0|0|0xa5a3dba744d3c6f1372f888f54447553|sw_a|EXTRACTS|12.Ab.34|789| hash:a5a3dba744d3c6f1372f888f54447553
	// 2023-10-07 12:00:00.01 MDT|1|001|0x9bd3989cf85b232ddadd73a1a312b249|sw_b|EXTRACTS|1.2.34|a.1.1| hash:9bd3989cf85b232ddadd73a1a312b249
	// 2023-10-07 12:00:00.02 MDT|1|002|0x7f0e8136c3aec6bbde74dfbad17aef1c|sw_a|EXTRACTS|abc123def| hash:7f0e8136c3aec6bbde74dfbad17aef1c
	// 2023-10-07 12:00:00.03 MDT|1|003|0x4907fb17a4212e2e09897fafa1cb758a|sw_a|EXTRACTS|127.0.0.1:8080|1|x20|X30| hash:4907fb17a4212e2e09897fafa1cb758a
	// 2023-10-07 12:00:00.04 MDT|1|004|0x1b7739c1e24d3a837e7821ecfb9a1be1|sw_a|EXTRACTS|3.cd|2|ABC.123_45|30| hash:1b7739c1e24d3a837e7821ecfb9a1be1
	// 2023-10-07 12:00:00.05 MDT|1|005|0x1b7739c1e24d3a837e7821ecfb9a1be1|sw_a|EXTRACTS|4.ef|3|DEF.678_90|40| hash:1b7739c1e24d3a837e7821ecfb9a1be1
	// 2023-10-07 12:00:00.06 MDT|1|006|0x1b7739c1e24d3a837e7821ecfb9a1be1|sw_a|EXTRACTS|5.gh|4|GHI.098_76|50| hash:1b7739c1e24d3a837e7821ecfb9a1be1
}
