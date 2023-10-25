# go-parser
Go-parser was written to support parsing of log files that were written for human consumption and are generally difficult to parse.

```
pauldunn@PAULs-14-MBP go-parser % go build         
pauldunn@PAULs-14-MBP go-parser % ./go-parser -help
Usage of ./go-parser: note that parsed output will be written to /Users/pauldunn/tmp/go-parser, using the data file name with '.parsed.txt' appended as a file suffix 
  -datafile string
    	Path to data file. Overrides input file DataDirectory.
  -inputfile string
    	Path to json file with inputs. See ./inputs/exampleInputs.json.
  -logfile string
    	Name of log file in /Users/pauldunn/tmp/go-parser; blank to print logs to terminal.
  -loglevel int
    	Logging level; default 1. Zero based index into: [debug info warning audit error] (default 1)
  -sqlcolumns int
    	When > 0, output parsed data as SQL INSERT INTO statements, instead of delimited data. The value specifies the maximum number of columns output in the VALUES clause.
  -sqldatatable string
    	Used with sqlColumnsPtr to specify the table in which to import pased data; the table should already exist. (default "data")
  -sqlhashtable string
    	Used with sqlColumnsPtr to specify the table in which to import the hash table; the table should already exist. (default "hash")
  -sqlite3file string
    	Fully qualified path to a sqlite3 database file that has tables already created. Output files will be imported into sqlite3 then deleted.
  -stdout
    	Output parsed data to STDOUT (in addition to file output)
  -threads int
    	Threads to use when processing a directory (default 6)
  -uniqueid string
    	Unique ID that is output with each parsed row.
  -uniqueidregex string
    	Regex that will be called on the input data to find a unique ID that is output with each parsed row. Overrides uniqueid parameter
```  

Features:
* Reading data - Supports reading from a file or directly from an from an io.Reader. Data and errors are returned via channels, allowing multi-threading. Data is returned via a channel, making iterating easy.
* Replacement - Supports direct replacement using regular expressions. This feature can be used to replace string lacking delimiters with strings that have delimiters, or for any other replacement purposes. Also supports replacement of date time strings with Unix epoch to save storage space. 
* Filtering - Supports both positive (line of data must match) and negative (line of data cannot match) filtering of data.
* Extraction - Supports "extraction". I.E. finding fields that match a regular expression, removing matches from input, and returning matches as an additional field. The main utility of extraction is when used with hashing to identify distinct row types. 
* Hashing - After extracting values from field(s) (columns of data), hash the field(s) in order to allow pareto analysis. I.E. If the input has two rows with a given field containing `some critical event flag=1` and `some critical event flag=2`, you may really only want to know how many events occurred with `some critical event flag`. By extracting the `flag` value and hashing the result, those two fields are now the same and a pareto can be built on hashes. Keep a map[hash]field so you can decode the hashes back to something meaningful. Also note that several columns may be able to be combined into a single hash. This can greatly reduce storage costs by replacing many text fields, that are frequently repeated, with a single hash that is smaller in size.
* Output directly to an Sqlite3 database.
* Output SQL INSERT INTO statements for direct insertion into a database. 

## Input
Inputs are supplied both with command line parameters, and an Inputs file that provides the parsing details specific to a type of input file. For details on Inputs see [parser.go](./parser/parser.go)
* A single input file can be processed by providing the `datafile` CLI parameter, which overrides Inputs.DataDirectory.
* No `datafile` CLI parameter and presence of a Inputs.ProcessedInputDirectory means to watch the Inputs.DataDirectory and process all files, forever. (Inputs.ProcessedInputDirectory is a directory, that if present, indicates to move processed input files that directory.)  
## Output
Output is written either to individual files, or an Sqlite3 database.
### Text output
Parsed output is written to <USER_HOME>/tmp/go-parser/<DATA_FILE_NAME>.parsed.txt; hashes are written to <USER_HOME>/tmp/go-parser/<DATA_FILE_NAME>.hashes.txt. While the output files are being written the suffix is ".locked". When the files are fully processed the ".locked" suffix is removed and callers can use the output files.    
### Sqlite3
Providing the input parameters `sqlite3datatable`, `sqlite3file`, `sqlite3hashtable` will cause the ouput to be directly written to an Sqlite3 database. 
### INSERT INTO
Providing the `sqlout` parameter causes the output to be written as SQL `INSERT INTO` statements. `VALUES` in the statements are quotes according to `Scanner.SqlQuoteColumns`. The assumption here is that the caller will create a database that with the expected fields, plus a enough NULLable string columns to accept the maximum number of extracts.

## Examples
For full working examples and additional documentation see [parser_test.go](./parser/parser_test.go)

You can also run the example app on one of the test files.

Example WITHOUT hashing:
<font size=0.5em>
```
go run ./go-parser.go -datafile="./parser/test/test_extract.txt" -inputfile="./inputs/exampleInput.json" -loglevel=0                
2023/10/11 18:28:16.251709 go-parser.go:77:   debug: user.Current(): &{Uid:501 Gid:20 Username:pauldunn Name:PAUL DUNN HomeDir:/Users/pauldunn}
2023/10/11 18:28:16.251742 go-parser.go:78:    info: Data and logs being saved to directory: /Users/pauldunn/tmp/go-parser
---------------- PARSED OUTPUT START ----------------
2023-10-07 12:00:00.00 MDT|0|0|notification|debug|multi word type|sw_a|Unit {} message ({})|EXTRACTS|12.Ab.34|789
2023-10-07 12:00:00.01 MDT|1|001|notification|info|SingleWordType|sw_b|Info SW version = {} release={}|EXTRACTS|1.2.34|a.1.1
2023-10-07 12:00:00.02 MDT|1|002|status|info|alphanumeric value|sw_a|Message with alphanumberic value {}|EXTRACTS|abc123def
2023-10-07 12:00:00.03 MDT|1|003|status|info|alphanumeric value|sw_a|val:{} flag:{} other:{} on {}|EXTRACTS|127.0.0.1:8080|1|x20|X30
2023-10-07 12:00:00.04 MDT|1|004|status|info|alphanumeric value|sw_a|val={} flag = {} other {} on ({})|EXTRACTS|3.cd|2|ABC.123_45|30
2023-10-07 12:00:00.05 MDT|1|005|status|info|alphanumeric value|sw_a|val={} flag = {} other {} on ({})|EXTRACTS|4.ef|3|DEF.678_90|40
2023-10-07 12:00:00.06 MDT|1|006|status|info|alphanumeric value|sw_a|val={} flag = {} other {} on ({})|EXTRACTS|5.gh|4|GHI.098_76|50
---------------- PARSED OUTPUT END   ----------------
2023/10/11 18:28:16.252592 go-parser.go:213:    info: total lines with unexpected number of fields=0
```
</font>

Example WITH hashing:
<font size=0.5em>
```
pauldunn@PAULs-14-MBP go-parser % go run ./go-parser.go -datafile="./parser/test/test_extract.txt" -inputfile="./inputs/exampleInputWithHashing.json" -sqlcolumns=27 -stdout -sqlhashtable=hashes -sqldatatable=parsed -sqlite3file=kill.db  -uniqueidregex="(?i)serial\\s+number\\s*:?\\s*(\\w+)" 
2023/10/24 19:32:30.206059 go-parser.go:136:    info: Data and logs being saved to directory: /Users/pauldunn/tmp/go-parser
2023/10/24 19:32:30.206480 go-parser.go:292:    info: parsed output file: /Users/pauldunn/tmp/go-parser/test_extract.txt.parsed.txtlocked
---------------- PARSED OUTPUT START ----------------
2023/10/24 19:32:30.206505 go-parser.go:318:    info: UniqueID found via regex: SOME_SERIAL
INSERT INTO parsed VALUES('SOME_SERIAL','2023-10-07 12:00:00.00 MDT',0,0,x'a07b3c1e3a1a0a0354fd900c1f38515d','12.Ab.34','789',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
INSERT INTO parsed VALUES('SOME_SERIAL','2023-10-07 12:00:00.01 MDT',1,001,x'11d590cff0915d91c47ee0cb22f33faa','1.2.34','a.1.1',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
INSERT INTO parsed VALUES('SOME_SERIAL','2023-10-07 12:00:00.02 MDT',1,002,x'2e7ddd79e7861f9157735943ba75e2b0','abc123def',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
INSERT INTO parsed VALUES('SOME_SERIAL','2023-10-07 12:00:00.03 MDT',1,003,x'03d287e66fa1648a82a312d09f998f53','127.0.0.1:8080','1','x20','X30',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
INSERT INTO parsed VALUES('SOME_SERIAL','2023-10-07 12:00:00.04 MDT',1,004,x'14a74c37f4ebbb911cd73aa6a00b7670','3.cd','2','ABC.123_45','30',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
INSERT INTO parsed VALUES('SOME_SERIAL','2023-10-07 12:00:00.05 MDT',1,005,x'14a74c37f4ebbb911cd73aa6a00b7670','4.ef','3','DEF.678_90','40',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
INSERT INTO parsed VALUES('SOME_SERIAL','2023-10-07 12:00:00.06 MDT',1,006,x'14a74c37f4ebbb911cd73aa6a00b7670','5.gh','4','GHI.098_76','50',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
---------------- PARSED OUTPUT END   ----------------
2023/10/24 19:32:30.206745 go-parser.go:378:    info: total lines with unexpected number of fields=0
2023/10/24 19:32:30.206792 go-parser.go:392:    info: hashes output file: /Users/pauldunn/tmp/go-parser/test_extract.txt.hashes.txtlocked
2023/10/24 19:32:30.206800 go-parser.go:400:    info: len(hashCounts)=5
2023/10/24 19:32:30.206802 go-parser.go:401:    info: Hashes and counts:
2023/10/24 19:32:30.206805 go-parser.go:404:    info: hash: x'14a74c37f4ebbb911cd73aa6a00b7670', count: 3, value: status|info|alphanumeric value|sw_a|val={} flag = {} other {} on ({})
2023/10/24 19:32:30.206860 go-parser.go:404:    info: hash: x'a07b3c1e3a1a0a0354fd900c1f38515d', count: 1, value: notification|debug|multi word type|sw_a|Unit {} message ({})
2023/10/24 19:32:30.206872 go-parser.go:404:    info: hash: x'11d590cff0915d91c47ee0cb22f33faa', count: 1, value: notification|info|SingleWordType|sw_b|Info SW version = {} release={}
2023/10/24 19:32:30.206880 go-parser.go:404:    info: hash: x'2e7ddd79e7861f9157735943ba75e2b0', count: 1, value: status|info|alphanumeric value|sw_a|Message with alphanumberic value {}
2023/10/24 19:32:30.206888 go-parser.go:404:    info: hash: x'03d287e66fa1648a82a312d09f998f53', count: 1, value: status|info|alphanumeric value|sw_a|val:{} flag:{} other:{} on {}
---------------- HASHED OUTPUT START   ----------------
INSERT INTO hashes VALUES(x'14a74c37f4ebbb911cd73aa6a00b7670', 'status|info|alphanumeric value|sw_a|val={} flag = {} other {} on ({})');
INSERT INTO hashes VALUES(x'a07b3c1e3a1a0a0354fd900c1f38515d', 'notification|debug|multi word type|sw_a|Unit {} message ({})');
INSERT INTO hashes VALUES(x'11d590cff0915d91c47ee0cb22f33faa', 'notification|info|SingleWordType|sw_b|Info SW version = {} release={}');
INSERT INTO hashes VALUES(x'2e7ddd79e7861f9157735943ba75e2b0', 'status|info|alphanumeric value|sw_a|Message with alphanumberic value {}');
INSERT INTO hashes VALUES(x'03d287e66fa1648a82a312d09f998f53', 'status|info|alphanumeric value|sw_a|val:{} flag:{} other:{} on {}');

---------------- HASHED OUTPUT END   ----------------
2023/10/24 19:32:30.211503 go-parser.go:447:   error: calling sqlite3: exit status 1, args: [kill.db]
2023/10/24 19:32:30.217470 go-parser.go:192:    info: go-parser processing complete...
</font>