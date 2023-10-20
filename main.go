// Author: Paul F. Dunn, https://github.com/paulfdunn/
// Original source location: https://github.com/paulfdunn/go-parser
// This code is licensed under the MIT license. Please keep this attribution when
// replicating/copying/reusing the code.
//
// This is a parsing application written using the go-parser package. Please see README.md
// for information and the test files for working examples.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/paulfdunn/go-parser/parser"
	"github.com/paulfdunn/logh"
)

const (
	appName = "go-parser"

	hashesOutputFileSuffix = ".hashes.txt"
	hashesOutputDelimiter  = "|"
	parsedOutputFileSuffix = ".parsed.txt"
)

var (
	lp  func(logh.LoghLevel, ...any)
	lpf func(logh.LoghLevel, string, ...any)

	// CLI flags
	dataFilePtr              *string
	inputFilePtr             *string
	logFilePtr               *string
	logLevel                 *int
	parsedOutputDelimiterPtr *string
	uniqueIdPtr              *string
	uniqueIdRegexPtr         *string
	stdoutPtr                *bool

	// dataDirectorySuffix is appended to the users home directory.
	dataDirectorySuffix = filepath.Join(`tmp`, appName)
	dataDirectory       string
)

func crashDetect() {
	if err := recover(); err != nil {
		errOut := fmt.Sprintf("panic: %+v\n%+v", err, string(debug.Stack()))
		fmt.Println(errOut)
		logh.Map[appName].Println(logh.Error, errOut)
		errShutdown := logh.ShutdownAll()
		if errShutdown != nil {
			logh.Map[appName].Printf(logh.Error, fmt.Sprintf("%#v", errShutdown))
		}
	}
}

func main() {
	defer crashDetect()

	usr, err := user.Current()
	if err != nil {
		fmt.Printf("Error getting user.Currrent: %+v", err)
	}

	dataDirectory = filepath.Join(usr.HomeDir, dataDirectorySuffix)
	err = os.MkdirAll(dataDirectory, 0777)
	if err != nil {
		fmt.Printf("Error creating data directory: : %+v", err)
	}

	dataFilePtr = flag.String("datafile", "", "Path to data file")
	uniqueIdPtr = flag.String("uniqueid", "", "Unique ID that is output with each parsed row.")
	uniqueIdRegexPtr = flag.String("uniqueidregex", "", "Regex that will be called on the input data to find a unique ID that "+
		"is output with each parsed row. Overrides uniqueid parameter")
	inputFilePtr = flag.String("inputfile", "", "Path to json file with inputs. See ./inputs/exampleInputs.json.")
	logFilePtr = flag.String("logfile", "", "Name of log file in "+dataDirectory+"; blank to print logs to terminal.")
	logLevel = flag.Int("loglevel", int(logh.Info), fmt.Sprintf("Logging level; default %d. Zero based index into: %v",
		int(logh.Info), logh.DefaultLevels))
	parsedOutputDelimiterPtr = flag.String("parseddelimiter", "|", "Delimiter used for parsed output.")
	stdoutPtr = flag.Bool("stdout", true, "Output parsed data to STDOUT (in addition to file output)")
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage of %s: note that parsed output will be written to %s, "+
			"using the data file name with '%s' appended as a file suffix \n",
			os.Args[0], dataDirectory, parsedOutputFileSuffix)
		flag.PrintDefaults()
	}
	flag.Parse()

	// Setup logging.
	var logFilepath string
	if *logFilePtr != "" {
		logFilepath = filepath.Join(dataDirectory, *logFilePtr)
	}
	logh.New(appName, logFilepath, logh.DefaultLevels, logh.LoghLevel(*logLevel),
		logh.DefaultFlags, 100, int64(100e6))
	lp = logh.Map[appName].Println
	lp(logh.Debug, "")
	lpf = logh.Map[appName].Printf
	lpf(logh.Debug, "user.Current(): %+v", usr)
	lpf(logh.Info, "Data and logs being saved to directory: %s", dataDirectory)

	parseFile(*inputFilePtr, *dataFilePtr)
}

// parseFile uses an input file from inputPath to process a data file from dataPath.
func parseFile(inputPath string, dataPath string) {
	inputs, err := parser.NewInputs(inputPath)
	if err != nil {
		lpf(logh.Error, "calling NewInputs: %s", err)
		os.Exit(7)
	}

	// Create the scanner and open the file.
	scnr, err := parser.NewScanner(*inputs)
	if err != nil {
		lpf(logh.Error, "calling NewScanner: %s", err)
		os.Exit(9)
	}
	err = scnr.OpenFileScanner(dataPath)
	if err != nil {
		lpf(logh.Error, "calling OpenScanner: %s", err)
		os.Exit(13)
	}

	// Process all data.
	processScanner(scnr, *inputs, dataPath)
	scnr.Shutdown()

	logh.ShutdownAll()
}

// processScanner takes a scanner, (optionally) finds the unique ID in the input to append to each row,
// then replaces, spits, extracts, and hashes all data from the scanner. The parsed data is
// saved to the output, and  hashes saved to a seperate file.
func processScanner(scnr *parser.Scanner, inputs parser.Inputs, dataPath string) {
	hashMap := make(map[string]string)
	hashCounts := make(map[string]int)

	dataChan, errorChan := scnr.Read(100, 100)

	parsedOutputFilePath := filepath.Join(dataDirectory, filepath.Base(dataPath)+parsedOutputFileSuffix)
	parsedOutputFile, err := os.Create(parsedOutputFilePath)
	lpf(logh.Info, "parsed output file: %s", parsedOutputFilePath)
	if err != nil {
		lpf(logh.Error, "calling os.Create: %s", err)
		os.Exit(17)
	}
	defer parsedOutputFile.Close()
	outputWriter := bufio.NewWriter(parsedOutputFile)
	defer outputWriter.Flush()

	hashing := false
	if inputs.HashColumns != nil && len(inputs.HashColumns) > 0 {
		hashing = true
	}

	var uniqueId string
	var uniqueIdRegex *regexp.Regexp
	unexpectedFieldCount := 0
	sortedHashColumns := sort.IntSlice(inputs.HashColumns)
	if *uniqueIdRegexPtr != "" {
		uniqueIdRegex = regexp.MustCompile(*uniqueIdRegexPtr)
	} else if *uniqueIdPtr != "" {
		uniqueId = *uniqueIdPtr
		lpf(logh.Info, "UniqueID from input: %s", uniqueId)
		uniqueId += *parsedOutputDelimiterPtr
	}

	if *stdoutPtr {
		fmt.Println("---------------- PARSED OUTPUT START ----------------")
	}

	for row := range dataChan {
		if uniqueId == "" && uniqueIdRegex != nil {
			match := uniqueIdRegex.FindStringSubmatch(row)
			if match != nil {
				uniqueId = match[1]
				lpf(logh.Info, "UniqueID found via regex: %s", uniqueId)
				uniqueId += *parsedOutputDelimiterPtr
			}
		}

		if scnr.Filter(row) {
			continue
		}

		// Replace, split, and extract.
		row = scnr.Replace(row)
		splits := scnr.Split(row)
		if len(splits) != inputs.ExpectedFieldCount {
			unexpectedFieldCount++
			lpf(logh.Error, "field count=%d, dataFile:%s, splits:%s", len(splits), *dataFilePtr, strings.Join(splits, *parsedOutputDelimiterPtr))
		}
		extracts, errors := scnr.Extract(splits)
		for _, errs := range errors {
			lpf(logh.Warning, "%+v", errs)
		}

		if hashing {
			sehc := splitsExcludeHashColumns(sortedHashColumns, splits, hashCounts, hashMap)
			out := uniqueId + strings.Join(sehc, *parsedOutputDelimiterPtr) + "|EXTRACTS|" + strings.Join(extracts, *parsedOutputDelimiterPtr)
			outputWriter.WriteString(out + "\n")
			if *stdoutPtr {
				fmt.Println(out)
			}
		} else {
			out := uniqueId + strings.Join(splits, *parsedOutputDelimiterPtr) + "|EXTRACTS|" + strings.Join(extracts, *parsedOutputDelimiterPtr)
			outputWriter.WriteString(out + "\n")
			if *stdoutPtr {
				fmt.Println(out)
			}
		}
	}

	lpf(logh.Info, "total lines with unexpected number of fields=%d", unexpectedFieldCount)
	for err := range errorChan {
		lp(logh.Error, err)
	}

	if *stdoutPtr {
		fmt.Println("---------------- PARSED OUTPUT END   ----------------")
	}

	if hashing {
		saveHashes(hashCounts, hashMap, dataPath)
	}
}

// saveHashes writes the hashes out to a file for later importing into a database.
func saveHashes(hashCounts map[string]int, hashMap map[string]string, dataPath string) {
	// Open output files
	hashesOutputFilePath := filepath.Join(dataDirectory, filepath.Base(dataPath)+hashesOutputFileSuffix)
	hashesOutputFile, err := os.Create(hashesOutputFilePath)
	lpf(logh.Info, "hashes output file: %s", hashesOutputFilePath)
	if err != nil {
		lpf(logh.Error, "calling os.Create: %s", err)
		os.Exit(17)
	}
	defer hashesOutputFile.Close()

	sortedHashKeys := parser.SortedHashMapCounts(hashCounts)
	lpf(logh.Info, "len(hashCounts)=%d", len(hashCounts))
	lpf(logh.Info, "Hashes and counts:")
	for _, v := range sortedHashKeys {
		lpf(logh.Info, "hash: %s, count: %d, value: %s", v, hashCounts[v], hashMap[v])
		out := strings.Join([]string{v, hashMap[v]}, hashesOutputDelimiter)
		_, err := hashesOutputFile.WriteString(out + "\n")
		if err != nil {
			lpf(logh.Error, "calling hashesOutputFile.WriteString: %s", err)
		}
	}
}

// splitsExcludeHashColumns creates a version of splits that doesn't included the hash columns.
// It also calculates the hash of splits and adds the hash to hashMap and hashCount
func splitsExcludeHashColumns(sortedHashColumns []int, splits []string,
	hashCounts map[string]int, hashMap map[string]string) []string {
	// Create the hash
	hashSplits := make([]string, 0, len(sortedHashColumns))
	for _, v := range sortedHashColumns {
		hashSplits = append(hashSplits, splits[v])
	}
	hashString := strings.Join(hashSplits, *parsedOutputDelimiterPtr)
	hash := "0x" + parser.Hash(hashString)
	hashMap[hash] = hashString
	hashCounts[hash] += 1

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

	return splitsExcludeHashColumns
}
