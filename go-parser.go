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
	"time"

	"github.com/paulfdunn/go-parser/parser"
	"github.com/paulfdunn/logh"
)

type flags struct {
	dataFilePath        string
	hashFormat          parser.HashFormat
	sqlite3FilePath     string
	sqlDataTable        string
	sqlHashTable        string
	sqlColumns          int
	stdout              bool
	threads             int
	uniqueId            string
	uniqueIdRegexString string
	uniqueIdRegex       *regexp.Regexp
}

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
	dataFilePtr      *string
	inputFilePtr     *string
	logFilePtr       *string
	logLevel         *int
	sqlite3FilePtr   *string
	sqlDataTablePtr  *string
	sqlHashTablePtr  *string
	sqlColumnsPtr    *int
	stdoutPtr        *bool
	threadsPtr       *int
	uniqueIdPtr      *string
	uniqueIdRegexPtr *string

	// dataDirectorySuffix is appended to the users home directory.
	dataDirectorySuffix = filepath.Join(`tmp`, appName)
	dataDirectory       string
)

func crashDetect() {
	if err := recover(); err != nil {
		errOut := fmt.Sprintf("panic: %+v\n%s", err, string(debug.Stack()))
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
		fmt.Printf("Error getting user.Currrent: %s", err)
	}

	dataDirectory = filepath.Join(usr.HomeDir, dataDirectorySuffix)
	err = os.MkdirAll(dataDirectory, 0777)
	if err != nil {
		fmt.Printf("Error creating data directory: : %s", err)
	}

	dataFilePtr = flag.String("datafile", "", "Path to data file. Overrides input file DataDirectory.")
	inputFilePtr = flag.String("inputfile", "", "Path to json file with inputs. See ./inputs/exampleInputs.json.")
	logFilePtr = flag.String("logfile", "", "Name of log file in "+dataDirectory+"; blank to print logs to terminal.")
	logLevel = flag.Int("loglevel", int(logh.Info), fmt.Sprintf("Logging level; default %d. Zero based index into: %v",
		int(logh.Info), logh.DefaultLevels))
	sqlite3FilePtr = flag.String("sqlite3file", "", "Fully qualified path to a sqlite3 database file that has tables already created. Output files will be imported into sqlite3 then deleted.")
	sqlDataTablePtr = flag.String("sqldatatable", "data", "Used with sqlColumnsPtr to specify the table in which to import pased data; the table should already exist.")
	sqlHashTablePtr = flag.String("sqlhashtable", "hash", "Used with sqlColumnsPtr to specify the table in which to import the hash table; the table should already exist.")
	sqlColumnsPtr = flag.Int("sqlcolumns", 0, "When > 0, output parsed data as SQL INSERT INTO statements, instead of delimited data. The value specifies the maximum number of columns output in the VALUES clause.")
	stdoutPtr = flag.Bool("stdout", false, "Output parsed data to STDOUT (in addition to file output)")
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

	hashFormat := parser.HASH_FORMAT_STRING
	if *sqlColumnsPtr > 0 {
		hashFormat = parser.HASH_FORMAT_SQL
	}
	flags := flags{
		dataFilePath:        *dataFilePtr,
		hashFormat:          hashFormat,
		sqlite3FilePath:     *sqlite3FilePtr,
		sqlDataTable:        *sqlDataTablePtr,
		sqlHashTable:        *sqlHashTablePtr,
		sqlColumns:          *sqlColumnsPtr,
		stdout:              *stdoutPtr,
		threads:             *threadsPtr,
		uniqueId:            *uniqueIdPtr,
		uniqueIdRegexString: *uniqueIdRegexPtr,
	}

	// The `datafile` CLI parameter overrides the Inputs.DataDirectory.
	if *dataFilePtr == "" && inputs.DataDirectory != "" {
		if _, err := os.Stat(inputs.DataDirectory); os.IsNotExist(err) {
			lpf(logh.Error, "inputs.DataDirectory does not exist: %s", err)
			os.Exit(5)
		}

		files, err := os.ReadDir(inputs.DataDirectory)
		if err != nil {
			lpf(logh.Error, "ReadDir error: %s", err)
			os.Exit(6)
		}

		// If inputs.ProcessedInputDirectory is empty, only process the DataDirectory once.
		// Otherwise watch the DataDirectory, forever.
		loops := 0
		for {
			parseFileEngine(inputs, files, flags)
			if inputs.ProcessedInputDirectory == "" {
				break
			}
			time.Sleep(time.Second)
			if loops%60 == 0 {
				lp(logh.Debug, "Waiting to process more input.")
			}
		}

	} else {
		parseFile(inputs, flags)
	}

	lpf(logh.Info, "%s processing complete...", appName)
	logh.ShutdownAll()
}

// parseFileEngine will use Go routines to start multiple instances of parseFile and process all
// files in the Inputs.DataDirectory.
func parseFileEngine(inputs *parser.Inputs, fileList []fs.DirEntry, flags flags) error {
	tasks := make(chan string, flags.threads)
	// Make sure the error buffer cannot fill up and cause a deadlock.
	// errorOut := make(chan error, threads)

	// Start number of Go Routines that will call s3mftDownloadFile
	var wg sync.WaitGroup
	for i := 0; i < flags.threads; i++ {
		wg.Add(1)
		go func() {
			for file := range tasks {
				flags.dataFilePath = file
				parseFile(inputs, flags)
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
func parseFile(inputs *parser.Inputs, flags flags) {

	// Create the scanner and open the file.
	scnr, err := parser.NewScanner(*inputs)
	if err != nil {
		lpf(logh.Error, "calling NewScanner: %s", err)
		os.Exit(9)
	}
	err = scnr.OpenFileScanner(flags.dataFilePath)
	if err != nil {
		lpf(logh.Error, "calling OpenScanner: %s", err)
		os.Exit(13)
	}

	// Process all data.
	parsedOutputFilePath := filepath.Join(dataDirectory, filepath.Base(flags.dataFilePath)+parsedOutputFileSuffix+lockedFileSuffix)
	hashesOutputFilePath := filepath.Join(dataDirectory, filepath.Base(flags.dataFilePath)+hashesOutputFileSuffix+lockedFileSuffix)
	processScanner(scnr, flags, parsedOutputFilePath, hashesOutputFilePath)
	scnr.Shutdown()

	// Rename the output files, removing the lockedFileSuffix
	parsedOutputFilePathUnlocked := filepath.Join(dataDirectory, filepath.Base(flags.dataFilePath)+parsedOutputFileSuffix)
	os.Rename(parsedOutputFilePath, parsedOutputFilePathUnlocked)
	hashesOutputFilePathUnlocked := filepath.Join(dataDirectory, filepath.Base(flags.dataFilePath)+hashesOutputFileSuffix)
	os.Rename(hashesOutputFilePath, hashesOutputFilePathUnlocked)

	if flags.sqlite3FilePath != "" {
		if scnr.HashingEnabled() && flags.sqlHashTable != "" {
			sqlite3Import(flags.sqlite3FilePath, hashesOutputFilePathUnlocked)
			os.Remove(hashesOutputFilePathUnlocked)
		}
		sqlite3Import(flags.sqlite3FilePath, parsedOutputFilePathUnlocked)
		os.Remove(parsedOutputFilePathUnlocked)
	}
}

// processScanner takes a scanner, (optionally) finds the unique ID in the input to append to each row,
// then replaces, spits, extracts, and hashes all data from the scanner. The parsed data is
// saved to the output, and  hashes saved to a seperate file.
func processScanner(scnr *parser.Scanner, flags flags, parsedOutputFilePath string, hashesOutputFilePath string) {

	dataChan, errorChan := scnr.Read(100, 100)

	parsedOutputFile, err := os.Create(parsedOutputFilePath)
	lpf(logh.Info, "parsed output file: %s", parsedOutputFilePath)
	if err != nil {
		lpf(logh.Error, "calling os.Create: %s", err)
		os.Exit(17)
	}
	defer parsedOutputFile.Close()
	outputWriter := bufio.NewWriter(parsedOutputFile)
	defer outputWriter.Flush()

	unexpectedFieldCount := 0
	if flags.uniqueIdRegexString != "" {
		flags.uniqueIdRegex = regexp.MustCompile(flags.uniqueIdRegexString)
	} else if *uniqueIdPtr != "" {
		lpf(logh.Info, "UniqueID from input: %s", flags.uniqueId)
		flags.uniqueId += scnr.OutputDelimiter
	}

	if flags.stdout {
		fmt.Println("---------------- PARSED OUTPUT START ----------------")
	}

	for row := range dataChan {
		if flags.uniqueId == "" && flags.uniqueIdRegex != nil {
			match := flags.uniqueIdRegex.FindStringSubmatch(row)
			if match != nil {
				flags.uniqueId = match[1]
				lpf(logh.Info, "UniqueID found via regex: %s", flags.uniqueId)
				flags.uniqueId += scnr.OutputDelimiter
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
		for _, err := range errors {
			lpf(logh.Warning, "%s", err)
		}

		if scnr.HashingEnabled() {
			sehc, err := scnr.SplitsExcludeHashColumns(splits, flags.hashFormat)
			if err != nil {
				lpf(logh.Error, "calling SplitsExcludeHashColumns: %s", err)
			}
			var out string

			if flags.sqlColumns > 0 {
				out = flags.uniqueId + scnr.SplitsToSql(flags.sqlColumns, flags.sqlDataTable, sehc, extracts)
			} else {
				out = flags.uniqueId + strings.Join(sehc, scnr.OutputDelimiter) + "|EXTRACTS|" + strings.Join(extracts, scnr.OutputDelimiter)
			}
			outputWriter.WriteString(out + "\n")
			if flags.stdout {
				fmt.Println(out)
			}
		} else {
			var out string
			if flags.sqlColumns > 0 {
				out = flags.uniqueId + scnr.SplitsToSql(flags.sqlColumns, flags.sqlDataTable, splits, extracts)
			} else {
				out = flags.uniqueId + strings.Join(splits, scnr.OutputDelimiter) + "|EXTRACTS|" + strings.Join(extracts, scnr.OutputDelimiter)
			}
			outputWriter.WriteString(out + "\n")
			if flags.stdout {
				fmt.Println(out)
			}
		}
	}

	if flags.stdout {
		fmt.Println("---------------- PARSED OUTPUT END   ----------------")
	}

	lpf(logh.Info, "total lines with unexpected number of fields=%d", unexpectedFieldCount)
	for err := range errorChan {
		lp(logh.Error, err)
	}

	if scnr.HashingEnabled() {
		saveHashes(scnr.HashCounts, scnr.HashMap, hashesOutputFilePath, flags)
	}
}

// saveHashes writes the hashes out to a file for later importing into a database.
func saveHashes(hashCounts map[string]int, hashMap map[string]string, hashesOutputFilePath string, flags flags) {
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
	var dump string
	for _, v := range sortedHashKeys {
		lpf(logh.Info, "hash: %s, count: %d, value: %s", v, hashCounts[v], hashMap[v])
		var out string
		if flags.sqlColumns > 0 {
			out = fmt.Sprintf("INSERT INTO %s VALUES(%s, '%s');", flags.sqlHashTable, v, hashMap[v]) + "\n"
		} else {
			out = strings.Join([]string{v, hashMap[v]}, hashesOutputDelimiter) + "\n"
		}
		dump += out
		_, err := hashesOutputFile.WriteString(out)
		if err != nil {
			lpf(logh.Error, "calling hashesOutputFile.WriteString: %s", err)
		}
	}

	if flags.stdout {
		fmt.Println("---------------- HASHED OUTPUT START   ----------------")
		fmt.Println(dump)
		fmt.Println("---------------- HASHED OUTPUT END   ----------------")
	}

}

// sqlite3Import is used to import the SQL output into a sqlite3 database.
// Sqlite3 will create the file if it does not exist. It is not great, as that means the user
// didn't create the table either. In which case, sqlite3 does the best it can and imports.
func sqlite3Import(sqlite3FilePath, outputFilePath string) {
	b, _ := os.ReadFile(outputFilePath)
	lpf(logh.Debug, string(b))
	// if _, err := os.Stat(sqlite3FilePath); err == nil {
	args := []string{sqlite3FilePath}
	sqc := fmt.Sprintf(".read %s", outputFilePath)
	cmd := exec.Command("sqlite3", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		lpf(logh.Error, "StdinPipe: %s", err)
	}
	_, err = io.WriteString(stdin, sqc)
	if err != nil {
		lpf(logh.Error, "WriteString: %s", err)
	}
	stdin.Close()
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		lpf(logh.Error, "calling sqlite3: %+v, args: %s", err, args)
	}
	lpf(logh.Debug, "stdoutStderr: \n%s", stdoutStderr)
	// } else {
	// 	lpf(logh.Error, "accessing sqlite3File path: %s", err)
	// }
}
