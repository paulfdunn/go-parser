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
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/paulfdunn/go-parser/parser"
	"github.com/paulfdunn/logh"
)

const (
	appName          = "go-parser"
	lockedFileSuffix = "locked"

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
	sqlite3FilePtr           *string
	sqlite3DataTablePtr      *string
	sqlite3HashTablePtr      *string
	stdoutPtr                *bool
	threadsPtr               *int
	uniqueIdPtr              *string
	uniqueIdRegexPtr         *string

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

	dataFilePtr = flag.String("datafile", "", "Path to data file. Overrides input file DataDirectory.")
	inputFilePtr = flag.String("inputfile", "", "Path to json file with inputs. See ./inputs/exampleInputs.json.")
	logFilePtr = flag.String("logfile", "", "Name of log file in "+dataDirectory+"; blank to print logs to terminal.")
	logLevel = flag.Int("loglevel", int(logh.Info), fmt.Sprintf("Logging level; default %d. Zero based index into: %v",
		int(logh.Info), logh.DefaultLevels))
	sqlite3FilePtr = flag.String("sqlite3file", "", "Fully qualified path to a sqlite3 database file that has tables already created. Output files will be imported into sqlite3 then deleted.")
	sqlite3DataTablePtr = flag.String("sqlite3datatable", "data", "Used with sqlite3file to specify the table in which to import pased data; the table should already exist.")
	sqlite3HashTablePtr = flag.String("sqlite3hashtable", "hash", "Used with sqlite3file to specify the table in which to import the hash table; the table should already exist.")
	stdoutPtr = flag.Bool("stdout", true, "Output parsed data to STDOUT (in addition to file output)")
	threadsPtr = flag.Int("threads", 6, "Threads to use when processing a directory")
	uniqueIdPtr = flag.String("uniqueid", "", "Unique ID that is output with each parsed row.")
	uniqueIdRegexPtr = flag.String("uniqueidregex", "", "Regex that will be called on the input data to find a unique ID that "+
		"is output with each parsed row. Overrides uniqueid parameter")
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

	inputs, err := parser.NewInputs(*inputFilePtr)
	if err != nil {
		lpf(logh.Error, "calling NewInputs: %s", err)
		os.Exit(7)
	}

	if *dataFilePtr == "" && inputs.DataDirectory != "" {
		if _, err := os.Stat(inputs.DataDirectory); os.IsNotExist(err) {
			lpf(logh.Error, "inputs.DataDirectory does not exist: %+v", err)
			os.Exit(5)
		}

		files, err := os.ReadDir(inputs.DataDirectory)
		if err != nil {
			lpf(logh.Error, "ReadDir error: %+v", err)
			os.Exit(6)
		}

		parseFileEngine(inputs, files, *threadsPtr, *sqlite3FilePtr, *sqlite3DataTablePtr, *sqlite3HashTablePtr)

	} else {
		parseFile(inputs, *dataFilePtr, *sqlite3FilePtr, *sqlite3DataTablePtr, *sqlite3HashTablePtr)
	}

	logh.ShutdownAll()
}

// parseFileEngine will use Go routines to start multiple instances of parseFile.
func parseFileEngine(inputs *parser.Inputs, fileList []fs.DirEntry, threads int,
	sqlite3FilePath string, sqlite3DataTable string, sqlite3HashTable string) error {
	tasks := make(chan string, threads)
	// Make sure the error buffer cannot fill up and cause a deadlock.
	// errorOut := make(chan error, threads)

	// Start number of Go Routines that will call s3mftDownloadFile
	var wg sync.WaitGroup
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			for file := range tasks {
				parseFile(inputs, file, sqlite3FilePath, sqlite3DataTable, sqlite3HashTable)
			}
			wg.Done()
		}()
	}

	// Read the download list, line by line, feeding work to the Go routines started above.
	for _, file := range fileList {
		fn := filepath.Join(inputs.DataDirectory, file.Name())
		lpf(logh.Debug, "calling parseFile for file: %s", fn)
		tasks <- fn

		// Need to prevent the error channel from filling up and blocking
		// DONE:
		// 	for {
		// 		select {
		// 		case e := <-errorOut:
		// 			lpf(logh.Error, "file download error: %+v", e)
		// 		default:
		// 			break DONE
		// 		}
		// 	}
	}
	close(tasks)

	// Wait for all work to be done and see if there were errors
	wg.Wait()
	// close(errorOut)
	// for e := range errorOut {
	// 	lpf(logh.Error, "file download error: %+v", e)
	// }

	return nil
}

// parseFile uses an input file from inputPath to process a data file from dataFilePath.
// While the output files are being written the suffix is ".locked". When the files are fully
// processed the ".locked" suffix is removed and callers can use the output files.
func parseFile(inputs *parser.Inputs, dataFilePath string, sqlite3FilePath string,
	sqlite3DataTable string, sqlite3HashTable string) {

	// Create the scanner and open the file.
	scnr, err := parser.NewScanner(*inputs)
	if err != nil {
		lpf(logh.Error, "calling NewScanner: %+v", err)
		os.Exit(9)
	}
	err = scnr.OpenFileScanner(dataFilePath)
	if err != nil {
		lpf(logh.Error, "calling OpenScanner: %+v", err)
		os.Exit(13)
	}

	// Process all data.
	parsedOutputFilePath := filepath.Join(dataDirectory, filepath.Base(dataFilePath)+parsedOutputFileSuffix+lockedFileSuffix)
	hashesOutputFilePath := filepath.Join(dataDirectory, filepath.Base(dataFilePath)+hashesOutputFileSuffix+lockedFileSuffix)
	processScanner(scnr, dataFilePath, parsedOutputFilePath, hashesOutputFilePath)
	scnr.Shutdown()

	// Rename the output files, removing the lockedFileSuffix
	parsedOutputFilePathUnlocked := filepath.Join(dataDirectory, filepath.Base(dataFilePath)+parsedOutputFileSuffix)
	os.Rename(parsedOutputFilePath, parsedOutputFilePathUnlocked)
	hashesOutputFilePathUnlocked := filepath.Join(dataDirectory, filepath.Base(dataFilePath)+hashesOutputFileSuffix)
	os.Rename(hashesOutputFilePath, hashesOutputFilePathUnlocked)

	if sqlite3FilePath != "" {
		if scnr.HashingEnabled() && sqlite3HashTable != "" {
			sqlite3Import(inputs.OutputDelimiter, sqlite3FilePath, hashesOutputFilePathUnlocked, sqlite3HashTable)
			os.Remove(hashesOutputFilePathUnlocked)
		}
		sqlite3Import(inputs.OutputDelimiter, sqlite3FilePath, parsedOutputFilePathUnlocked, sqlite3DataTable)
		os.Remove(parsedOutputFilePathUnlocked)
	}
}

// processScanner takes a scanner, (optionally) finds the unique ID in the input to append to each row,
// then replaces, spits, extracts, and hashes all data from the scanner. The parsed data is
// saved to the output, and  hashes saved to a seperate file.
func processScanner(scnr *parser.Scanner, dataFilePath string,
	parsedOutputFilePath string, hashesOutputFilePath string) {

	dataChan, errorChan := scnr.Read(100, 100)

	parsedOutputFile, err := os.Create(parsedOutputFilePath)
	lpf(logh.Info, "parsed output file: %s", parsedOutputFilePath)
	if err != nil {
		lpf(logh.Error, "calling os.Create: %+v", err)
		os.Exit(17)
	}
	defer parsedOutputFile.Close()
	outputWriter := bufio.NewWriter(parsedOutputFile)
	defer outputWriter.Flush()

	var uniqueId string
	var uniqueIdRegex *regexp.Regexp
	unexpectedFieldCount := 0
	if *uniqueIdRegexPtr != "" {
		uniqueIdRegex = regexp.MustCompile(*uniqueIdRegexPtr)
	} else if *uniqueIdPtr != "" {
		uniqueId = *uniqueIdPtr
		lpf(logh.Info, "UniqueID from input: %s", uniqueId)
		uniqueId += scnr.OutputDelimiter
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
				uniqueId += scnr.OutputDelimiter
			}
		}

		if scnr.Filter(row) {
			continue
		}

		// Replace, split, and extract.
		row = scnr.Replace(row)
		splits, err := scnr.Split(row)
		if err != nil {
			unexpectedFieldCount++
			lpf(logh.Error, "%+v, splits:%s", err, strings.Join(splits, scnr.OutputDelimiter))
		}
		extracts, errors := scnr.Extract(splits)
		for _, errs := range errors {
			lpf(logh.Warning, "%+v", errs)
		}

		if scnr.HashingEnabled() {
			sehc := scnr.SplitsExcludeHashColumns(splits)
			out := uniqueId + strings.Join(sehc, scnr.OutputDelimiter) + "|EXTRACTS|" + strings.Join(extracts, scnr.OutputDelimiter)
			outputWriter.WriteString(out + "\n")
			if *stdoutPtr {
				fmt.Println(out)
			}
		} else {
			out := uniqueId + strings.Join(splits, scnr.OutputDelimiter) + "|EXTRACTS|" + strings.Join(extracts, scnr.OutputDelimiter)
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

	if scnr.HashingEnabled() {
		saveHashes(scnr.HashCounts, scnr.HashMap, hashesOutputFilePath, dataFilePath)
	}
}

// saveHashes writes the hashes out to a file for later importing into a database.
func saveHashes(hashCounts map[string]int, hashMap map[string]string, hashesOutputFilePath string, dataFilePath string) {
	// Open output files
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

// sqlite3Import is used to import the delimited output files into a sqlite3 database.
// Sqlite3 will create the file if it does not exist. It is not great, as that means the user
// didn't create the table either. In which case, sqlite3 does the best it can and imports.
func sqlite3Import(outputDelimiter, sqlite3FilePath, outputFilePath, sqlite3Table string) {
	// if _, err := os.Stat(sqlite3FilePath); err == nil {
	args := []string{"-separator", outputDelimiter, sqlite3FilePath}
	sqc := fmt.Sprintf(".import %s %s", outputFilePath, sqlite3Table)
	cmd := exec.Command("sqlite3", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		lpf(logh.Error, "StdinPipe: %+v", err)
	}
	_, err = io.WriteString(stdin, sqc)
	if err != nil {
		lpf(logh.Error, "WriteString: %+v", err)
	}
	stdin.Close()
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		lpf(logh.Error, "calling sqlite3: %+v, args: %+v", err, args)
	}
	lpf(logh.Debug, "stdoutStderr: \n%s", stdoutStderr)
	// } else {
	// 	lpf(logh.Error, "accessing sqlite3File path: %+v", err)
	// }
}
