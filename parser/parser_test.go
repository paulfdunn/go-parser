package parser

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
)

const (
	testDataDirectory = "./test"
)

// openFileScanner is a convenience test function for getting a NewScanner to the test file.
// Callers must `defer scnr.Shutdown()`
func openFileScanner(testDataFilePath string, negativeFilter string, positiveFilter string,
	delimiter string, replacement []*Replacement, extract []*Extract) *Scanner {
	scnr, err := NewScanner(negativeFilter, positiveFilter, delimiter, replacement, extract)
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
	scnr, err := NewScanner("", "", "", []*Replacement{}, []*Extract{})
	if err != nil {
		var t *testing.T
		t.Errorf("calling OpenScanner: %s", err)
	}
	scnr.OpenFileScanner(path.Join(testDataDirectory, "test_read.txt"))
	defer scnr.Shutdown()

	//Output:
}

// ExampleScanner_OpenIoReaderScanner shows how to open an io.Reader for processing.
// Note that a file is used for convenience in calling OpenIoReaderScanner. When
// processing files, use the OpenFileScanner convenience function.
func ExampleScanner_OpenIoReaderScanner() {
	file, err := os.Open(path.Join(testDataDirectory, "test_read.txt"))
	if err != nil {
		var t *testing.T
		t.Errorf("calling os.Open: %s", err)
	}

	scnr, err := NewScanner("", "", "", []*Replacement{}, []*Extract{})
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
	scnr := openFileScanner(path.Join(testDataDirectory, "test_read.txt"), "", "", "", []*Replacement{}, []*Extract{})
	fmt.Println("Read all the test data")
	dataChan, errorChan := scnr.Read(100, 100)
	for row := range dataChan {
		fmt.Println(row)
	}
	for err := range errorChan {
		fmt.Println(err)
	}

	scnr.Shutdown()

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
	scnr := openFileScanner(path.Join(testDataDirectory, "test_filter.txt"), `#|negative\s+filter`,
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

	scnr.Shutdown()

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
	scnr := openFileScanner(path.Join(testDataDirectory, "test_filter.txt"), "",
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

	scnr.Shutdown()

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
	scnr := openFileScanner(path.Join(testDataDirectory, "test_replace.txt"), "", "", delimiter, rplc, []*Extract{})
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

	scnr.Shutdown()

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
	scnr := openFileScanner(path.Join(testDataDirectory, "test_read.txt"), "", "",
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

	scnr.Shutdown()

	// Output:
	//
	// Input data:
	// 2023-10-07 12:00:00.00 MDT  0         0         notification  debug          multi word type     sw_a          Debug SW message
	// 2023-10-07 12:00:00.01 MDT  1         001       notification  info           SingleWordType      sw_b          Info SW message
	// 2023-10-07 12:00:00.02 MDT  1         002       status        info           alphanumeric value  sw_a          Message with alphanumberic value abc123def
	//
	// Split data:
	// 2023-10-07 12:00:00.00 MDT|0|0|notification|debug|multi word type|sw_a|Debug SW message
	// 2023-10-07 12:00:00.01 MDT|1|001|notification|info|SingleWordType|sw_b|Info SW message
	// 2023-10-07 12:00:00.02 MDT|1|002|status|info|alphanumeric value|sw_a|Message with alphanumberic value abc123def
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
		Column:      7,
		RegexString: `(^|\s+)(([0-9]+[a-zA-Z_\.\-\:]|[a-zA-Z]+[0-9_\.\-\:])[a-zA-Z0-9_\.\-\:]*)(\s+|$)`,
		Token:       " {} ",
		Submatch:    2}
	extrct2 := Extract{
		Column:      7,
		RegexString: `(^|\s+)([0-9\.]+)(^|\s+)`,
		Token:       " {} ",
		Submatch:    2}
	extrct3 := Extract{
		Column:      7,
		RegexString: `=([0-9\.]+)(^|\s+)`,
		Token:       "={} ",
		Submatch:    1}
	scnr := openFileScanner(path.Join(testDataDirectory, "test_extract.txt"), "", "",
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
			"| hash:"+Hash(splits[7]))
	}
	for err := range errorChan {
		fmt.Println(err)
	}

	fmt.Println("\nInput data:")
	fmt.Printf("%+v", strings.Join(fullData, "\n"))
	fmt.Println("\n\nExtract(ed) data:")
	fmt.Printf("%+v", strings.Join(extractData, "\n"))

	scnr.Shutdown()

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
	// 2023-10-07 12:00:00.00 MDT|0|0|notification|debug|multi word type|sw_a|Unit {} message| extracts:12.Ab.34| hash:7845ba21efbbc5c7e56f9e03c16d9fc9
	// 2023-10-07 12:00:00.01 MDT|1|001|notification|info|SingleWordType|sw_b|Info SW version = {} message| extracts:1.2.34| hash:083e30752d07c3bf565f8c12de3c5372
	// 2023-10-07 12:00:00.02 MDT|1|002|status|info|alphanumeric value|sw_a|Message with alphanumberic value {} | extracts:abc123def| hash:eba72bd27dfbbec0b24ef6f15b558bd8
	// 2023-10-07 12:00:00.03 MDT|1|003|status|info|alphanumeric value|sw_a|val={} flag = {} other {} | extracts:3.AB|20|1| hash:66c15fc3978f76f1db989d1caadfeae9
	// 2023-10-07 12:00:00.03 MDT|1|003|status|info|alphanumeric value|sw_a|val={} flag = {} other {} | extracts:4.cd|30|2| hash:66c15fc3978f76f1db989d1caadfeae9
}
