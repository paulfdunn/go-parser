# go-parser
Go-parser was written to support parsing of log files that were written for human consumption and are generally difficult to parse.

Features:
* Reading data - Supports reading from a file or directly from an from an io.Reader. Data and errors are returned via channels, allowing multi-threading. Data is returned via a channel, making iterating easy.
* Replacement - Supports direct replacement using regular expressions. This feature can be used to replace string lacking delimiters with strings that have delimiters, or for any other replacement purposes.
* Filtering - Supports both positive (line of data must match) and negative (line of data cannot match) filtering of data.
* Extraction - Supports "extraction". I.E. finding fields that match a regular expression, removing matches from input, and returning matches as an additional field. The main utility of extraction is when used with hashing to identify distinct row types. 
* Hashing - After extracting values from field(s) (columns of data), hash the field(s) in order to allow pareto analysis. I.E. If the input has two rows with a given field containing `some critical event flag=1` and `some critical event flag=2`, you may really only want to know how many events occurred with `some critical event flag`. By extracting the `flag` value and hashing the result, those two fields are now the same and a pareto can be built on hashes. Keep a map[hash]field so you can decode the hashes back to something meaningful. Also note that several columns may be able to be combined into a single hash. This can greatly reduce storage costs by replacing many text fields, that are frequently repeated, with a single hash that is smaller in size.

## Output
Parsed output is written to <USER_HOME>/tmp/go-parser/<DATA_FILE_NAME>.parsed.txt; hashes are written to <USER_HOME>/tmp/go-parser/<DATA_FILE_NAME>.hashes.txt    

## Examples
For full working examples and additional documentation see [parser_test.go](./parser/parser_test.go)

You can also run the example app on one of the test files.

Example WITHOUT hashing:
<font size=0.5em>
```
go run ./main.go -datafile="./parser/test/test_extract.txt" -inputfile="./inputs/exampleInput.json" -loglevel=0                
2023/10/11 18:28:16.251709 main.go:77:   debug: user.Current(): &{Uid:501 Gid:20 Username:pauldunn Name:PAUL DUNN HomeDir:/Users/pauldunn}
2023/10/11 18:28:16.251742 main.go:78:    info: Data and logs being saved to directory: /Users/pauldunn/tmp/go-parser
---------------- PARSED OUTPUT START ----------------
2023-10-07 12:00:00.00 MDT|0|0|notification|debug|multi word type|sw_a|Unit {} message ({})|EXTRACTS|12.Ab.34|789
2023-10-07 12:00:00.01 MDT|1|001|notification|info|SingleWordType|sw_b|Info SW version = {} release={}|EXTRACTS|1.2.34|a.1.1
2023-10-07 12:00:00.02 MDT|1|002|status|info|alphanumeric value|sw_a|Message with alphanumberic value {}|EXTRACTS|abc123def
2023-10-07 12:00:00.03 MDT|1|003|status|info|alphanumeric value|sw_a|val:{} flag:{} other:{} on {}|EXTRACTS|127.0.0.1:8080|1|x20|X30
2023-10-07 12:00:00.04 MDT|1|004|status|info|alphanumeric value|sw_a|val={} flag = {} other {} on ({})|EXTRACTS|3.cd|2|ABC.123_45|30
2023-10-07 12:00:00.05 MDT|1|005|status|info|alphanumeric value|sw_a|val={} flag = {} other {} on ({})|EXTRACTS|4.ef|3|DEF.678_90|40
2023-10-07 12:00:00.06 MDT|1|006|status|info|alphanumeric value|sw_a|val={} flag = {} other {} on ({})|EXTRACTS|5.gh|4|GHI.098_76|50
---------------- PARSED OUTPUT END   ----------------
2023/10/11 18:28:16.252592 main.go:213:    info: total lines with unexpected number of fields=0
```
</font>

Example WITH hashing:
<font size=0.5em>
```
go run ./main.go -datafile="./parser/test/test_extract.txt" -inputfile="./inputs/exampleInputWithHashing.json" -loglevel=0
2023/10/11 18:28:37.498537 main.go:77:   debug: user.Current(): &{Uid:501 Gid:20 Username:pauldunn Name:PAUL DUNN HomeDir:/Users/pauldunn}
2023/10/11 18:28:37.498574 main.go:78:    info: Data and logs being saved to directory: /Users/pauldunn/tmp/go-parser
---------------- PARSED OUTPUT START ----------------
2023-10-07 12:00:00.00 MDT|0|0|a07b3c1e3a1a0a0354fd900c1f38515d|EXTRACTS|12.Ab.34|789
2023-10-07 12:00:00.01 MDT|1|001|11d590cff0915d91c47ee0cb22f33faa|EXTRACTS|1.2.34|a.1.1
2023-10-07 12:00:00.02 MDT|1|002|2e7ddd79e7861f9157735943ba75e2b0|EXTRACTS|abc123def
2023-10-07 12:00:00.03 MDT|1|003|03d287e66fa1648a82a312d09f998f53|EXTRACTS|127.0.0.1:8080|1|x20|X30
2023-10-07 12:00:00.04 MDT|1|004|14a74c37f4ebbb911cd73aa6a00b7670|EXTRACTS|3.cd|2|ABC.123_45|30
2023-10-07 12:00:00.05 MDT|1|005|14a74c37f4ebbb911cd73aa6a00b7670|EXTRACTS|4.ef|3|DEF.678_90|40
2023-10-07 12:00:00.06 MDT|1|006|14a74c37f4ebbb911cd73aa6a00b7670|EXTRACTS|5.gh|4|GHI.098_76|50
---------------- PARSED OUTPUT END   ----------------
2023/10/11 18:28:37.499420 main.go:207:    info: len(hashCounts)=5
2023/10/11 18:28:37.499427 main.go:208:   debug: Hashes and counts:
2023/10/11 18:28:37.499433 main.go:210:   debug: hash: 14a74c37f4ebbb911cd73aa6a00b7670, count: 3, value: status|info|alphanumeric value|sw_a|val={} flag = {} other {} on ({})
2023/10/11 18:28:37.499441 main.go:210:   debug: hash: 03d287e66fa1648a82a312d09f998f53, count: 1, value: status|info|alphanumeric value|sw_a|val:{} flag:{} other:{} on {}
2023/10/11 18:28:37.499446 main.go:210:   debug: hash: a07b3c1e3a1a0a0354fd900c1f38515d, count: 1, value: notification|debug|multi word type|sw_a|Unit {} message ({})
2023/10/11 18:28:37.499451 main.go:210:   debug: hash: 11d590cff0915d91c47ee0cb22f33faa, count: 1, value: notification|info|SingleWordType|sw_b|Info SW version = {} release={}
2023/10/11 18:28:37.499456 main.go:210:   debug: hash: 2e7ddd79e7861f9157735943ba75e2b0, count: 1, value: status|info|alphanumeric value|sw_a|Message with alphanumeric value {}
2023/10/11 18:28:37.499460 main.go:213:    info: total lines with unexpected number of fields=0
```
</font>