package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/paulfdunn/go-parser/parser"
	"github.com/paulfdunn/logh"
)

var (
	appName = "go-parser"

	// CLI flags
	dataFilePtr  *string
	inputFilePtr *string
	logFilePtr   *string
	logLevel     *int

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

type Inputs struct {
	Delimiter          string
	DelimiterString    string
	ExpectedFieldCount int
	Extracts           []*parser.Extract
	HashColumns        []int
	NegativeFilter     string
	PositiveFilter     string
	ProcessedDirectory string
	Replacements       []*parser.Replacement
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
	inputFilePtr = flag.String("inputfile", "", "Path to json file with inputs. See exampleInputs.json.")
	logFilePtr = flag.String("logfile", "", "Name of log file in "+dataDirectory+"; blank to print logs to terminal.")
	logLevel = flag.Int("loglevel", int(logh.Info), fmt.Sprintf("Logging level; default %d. Zero based index into: %v",
		int(logh.Info), logh.DefaultLevels))
	flag.Parse()

	// Setup logging.
	var logFilepath string
	if *logFilePtr != "" {
		logFilepath = filepath.Join(dataDirectory, *logFilePtr)
	}
	logh.New(appName, logFilepath, logh.DefaultLevels, logh.LoghLevel(*logLevel),
		logh.DefaultFlags, 100, int64(10e6))
	lp := logh.Map[appName].Println
	lpf := logh.Map[appName].Printf
	lpf(logh.Debug, "user.Current(): %+v", usr)
	lpf(logh.Info, "Data and logs being saved to directory: %s", dataDirectory)

	// Process the input file.
	inputBytes, err := os.ReadFile(*inputFilePtr)
	if err != nil {
		lpf(logh.Error, "opening JSON input file: %+v", err)
		logh.ShutdownAll()
		os.Exit(1)
	}
	inputs := Inputs{}
	err = json.Unmarshal(inputBytes, &inputs)
	if err != nil {
		lpf(logh.Error, "unmarshaling JSON input file: %+v", err)
		logh.ShutdownAll()
		os.Exit(5)
	}

	// Create the scanner and open the file.
	scnr, err := parser.NewScanner(inputs.NegativeFilter, inputs.PositiveFilter,
		inputs.Delimiter, inputs.Replacements, inputs.Extracts, inputs.ProcessedDirectory)
	if err != nil {
		lpf(logh.Error, "calling NewScanner: %s", err)
		os.Exit(9)
	}
	err = scnr.OpenFileScanner(*dataFilePtr)
	if err != nil {
		lpf(logh.Error, "calling OpenScanner: %s", err)
		os.Exit(13)
	}

	// Process all data.
	dataChan, errorChan := scnr.Read(100, 100)
	rows := 0
	unexpectedFieldCount := 0
	hashMap := make(map[string]string)
	hashCounts := make(map[string]int)
	sortedHashColumns := sort.IntSlice(inputs.HashColumns)
	hashing := false
	if inputs.HashColumns != nil && len(inputs.HashColumns) > 0 {
		hashing = true
	}
	for row := range dataChan {
		if scnr.Filter(row) {
			continue
		}

		// Replace, split, and extract.
		row = scnr.Replace(row)
		splits := scnr.Split(row)
		if len(splits) != inputs.ExpectedFieldCount {
			unexpectedFieldCount++
			lp(logh.Error, fmt.Sprintf("field count=%d|", len(splits))+strings.Join(splits, "|"))
		}
		extracts := scnr.Extract(splits)

		if hashing {
			// Create the hash
			hashSplits := make([]string, 0, len(sortedHashColumns))
			for _, v := range sortedHashColumns {
				hashSplits = append(hashSplits, splits[v])
			}
			hashString := strings.Join(hashSplits, "|")
			hash := parser.Hash(hashString)
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

			lp(logh.Info, strings.Join(splitsExcludeHashColumns, "|")+"| extracts:"+strings.Join(extracts, "|"))
		} else {
			lp(logh.Info, strings.Join(splits, "|")+"| extracts:"+strings.Join(extracts, "|"))
		}

		rows++
	}
	scnr.Shutdown()

	for err := range errorChan {
		lp(logh.Error, err)
	}

	if hashing {
		sortedHashKeys := parser.SortedHashMapCounts(hashCounts)
		lpf(logh.Info, "len(hashCounts)=%d", len(hashCounts))
		lpf(logh.Debug, "Hashes and counts:")
		for _, v := range sortedHashKeys {
			lpf(logh.Debug, "hash: %s, count: %d, value: %s", v, hashCounts[v], hashMap[v])
		}
	}
	lpf(logh.Info, "total lines with unexpected number of fields=%d", unexpectedFieldCount)
	logh.ShutdownAll()
}
