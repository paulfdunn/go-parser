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
func openFileScanner(testDataFilePath string, negativeFilter string, positiveFilter string,
	delimiter string, replacement []*Replacement, extract []*Extract) *Scanner {
	scnr, err := NewScanner(negativeFilter, positiveFilter, delimiter, replacement, extract, "")
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
	scnr, err := NewScanner("", "", "", []*Replacement{}, []*Extract{}, "")
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

	scnr, err := NewScanner("", "", "", []*Replacement{}, []*Extract{}, "")
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
	scnr := openFileScanner(filepath.Join(testDataDirectory, "test_read.txt"), "", "", "", []*Replacement{}, []*Extract{})
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

	scnr := openFileScanner(tmpInputFilePath, "", "", "", []*Replacement{}, []*Extract{})
	scnr.processedDirectory = t.TempDir()
	fmt.Println("Read all the test data")
	dataChan, errorChan := scnr.Read(100, 100)
	for row := range dataChan {
		fmt.Println(row)
	}
	for err := range errorChan {
		fmt.Println(err)
	}

	// Read the file from the processedDirectory and byte compare to the original
	testFileBytesAfterMove, err := os.ReadFile(filepath.Join(scnr.processedDirectory, testFileName))
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
	scnr := openFileScanner(filepath.Join(testDataDirectory, "test_filter.txt"), `#|negative\s+filter`,
		"", "", []*Replacement{}, []*Extract{})
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
	scnr := openFileScanner(filepath.Join(testDataDirectory, "test_filter.txt"), "",
		`\d{4}-\d{2}-\d{2}[ -]\d{2}:\d{2}:\d{2}\.\d{2}\s+[a-zA-Z]{2,5}`,
		"", []*Replacement{}, []*Extract{})
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
	scnr := openFileScanner(filepath.Join(testDataDirectory, "test_replace.txt"), "", "", delimiter, rplc, []*Extract{})
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
func ExampleScanner_Split() {
	delimiter := `\s\s+`
	delimiterString := "  "
	scnr := openFileScanner(filepath.Join(testDataDirectory, "test_split.txt"), "", "",
		delimiter, []*Replacement{{RegexString: `\s\s+`, Replacement: delimiterString}}, []*Extract{})
	dataChan, errorChan := scnr.Read(100, 100)
	fullData := []string{}
	splitData := []string{}
	for row := range dataChan {
		fullData = append(fullData, row)
		splits := scnr.Split(row)
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
	extrct1 := Extract{
		Columns:     []int{7},
		RegexString: `(^|\s+)(([0-9]+[a-zA-Z_\.\-\:]|[a-zA-Z]+[0-9_\.\-\:])[a-zA-Z0-9_\.\-\:]*)(\s+|$)`,
		Token:       " {} ",
		Submatch:    2}
	extrct2 := Extract{
		Columns:     []int{7},
		RegexString: `(^|\s+)([0-9\.]+)(^|\s+)`,
		Token:       " {} ",
		Submatch:    2}
	extrct3 := Extract{
		Columns:     []int{7},
		RegexString: `=([0-9\.]+)(^|\s+)`,
		Token:       "={} ",
		Submatch:    1}
	scnr := openFileScanner(filepath.Join(testDataDirectory, "test_extract.txt"), "", "",
		delimiter, []*Replacement{{RegexString: `\s\s+`, Replacement: delimiterString}},
		[]*Extract{&extrct1, &extrct2, &extrct3})
	dataChan, errorChan := scnr.Read(100, 100)
	fullData := []string{}
	extractData := []string{}
	for row := range dataChan {
		splits := scnr.Split(row)
		fullData = append(fullData, strings.Join(splits, "|"))
		extracts := scnr.Extract(splits)
		extractData = append(extractData, strings.Join(splits, "|")+
			"| extracts:"+strings.Join(extracts, "|")+
			"| hash:"+Hash(splits[3]+splits[4]+splits[5]+splits[7]))
	}
	for err := range errorChan {
		fmt.Println(err)
	}

	fmt.Println("\nInput data:")
	fmt.Printf("%+v", strings.Join(fullData, "\n"))
	fmt.Println("\n\nExtract(ed) data:")
	fmt.Printf("%+v", strings.Join(extractData, "\n"))

	// Output:
	//
	// Input data:
	// 2023-10-07 12:00:00.00 MDT|0|0|notification|debug|multi word type|sw_a|Unit 12.Ab.34 message
	// 2023-10-07 12:00:00.01 MDT|1|001|notification|info|SingleWordType|sw_b|Info SW version = 1.2.34 message
	// 2023-10-07 12:00:00.02 MDT|1|002|status|info|alphanumeric value|sw_a|Message with alphanumberic value abc123def
	// 2023-10-07 12:00:00.03 MDT|1|003|status|info|alphanumeric value|sw_a|val=1 flag = 20 other 3.AB
	// 2023-10-07 12:00:00.03 MDT|1|003|status|info|alphanumeric value|sw_a|val=2 flag = 30 other 4.cd
	//
	// Extract(ed) data:
	// 2023-10-07 12:00:00.00 MDT|0|0|notification|debug|multi word type|sw_a|Unit {} message| extracts:12.Ab.34| hash:74ed0ffb5be98f93d2b9a2ca360e5ac3
	// 2023-10-07 12:00:00.01 MDT|1|001|notification|info|SingleWordType|sw_b|Info SW version = {} message| extracts:1.2.34| hash:c7ea26ece6e34c6763e7df04341868ca
	// 2023-10-07 12:00:00.02 MDT|1|002|status|info|alphanumeric value|sw_a|Message with alphanumberic value {} | extracts:abc123def| hash:b60cbe35cb07181c41758f3a1a6bb263
	// 2023-10-07 12:00:00.03 MDT|1|003|status|info|alphanumeric value|sw_a|val={} flag = {} other {} | extracts:3.AB|20|1| hash:4cf1e01d8894ae26497d87b76f126ce9
	// 2023-10-07 12:00:00.03 MDT|1|003|status|info|alphanumeric value|sw_a|val={} flag = {} other {} | extracts:4.cd|30|2| hash:4cf1e01d8894ae26497d87b76f126ce9
}
