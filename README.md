# go-parser
Go-parser was written to support parsing of log files that were written for human consumption and are generally difficult to parse.

Features:
* Reading data - Supports reading from a file or directly from an from an io.Reader. Data and errors are returned via channels, allowing multi-threading. Data is returned via a channel, making iterating easy.
* Replacement - Supports direct replacement using regular expressions. This feature can be used to replace string lacking delimiters with strings that have delimiters, or for any other replacement purposes.
* Filtering - Supports both positive (line of data must match) and negative (line of data cannot match) filtering of data.
* Extraction - Supports "extraction". I.E. finding fields that match a regular expression, removing matches from input, and returning matches as an additional field. The main utility of extraction is when used with hashing to identify distinct row types. 
* Hashing - After extracting values from field(s) (columns of data), hash the field(s) in order to allow pareto analysis. I.E. If the input has two rows with a given field containing `some critical event flag=1` and `some critical event flag=2`, you may really only want to know how many events occured with `some critical event flag`. By extracting the `flag` value and hashing the result, those two fields are now the same and a pareto can be built on hashes. Keep a map[hash]field so you can decode the hashes back to something meaningful. Also note that several columns may be able to be combined into a single hash. This can greatly reduce storage costs by replacing many text fields, that are frequently repeated, with a single hash that is smaller in size.

For full working examples and additional documentation see [parser_test.go](./parser/parser_test.go)

You can also run the example app on one of the test files.

Example WITHOUT hashing:
<font size=0.5em>
```
go run ./main.go -datafile="./parser/test/test_extract.txt" -inputfile="exampleInput.json" -loglevel=0
2023/10/11 18:13:17.526453 main.go:77:   debug: user.Current(): &{Uid:501 Gid:20 Username:pauldunn Name:PAUL DUNN HomeDir:/Users/pauldunn}
2023/10/11 18:13:17.526490 main.go:78:    info: Data and logs being saved to directory: /Users/pauldunn/tmp/go-parser
---------------- PARSED OUTPUT START ----------------
2023-10-07 12:00:00.00 MDT|0|0|notification|debug|multi word type|sw_a|Unit {} message ({})| extracts:12.Ab.34|789
2023-10-07 12:00:00.01 MDT|1|001|notification|info|SingleWordType|sw_b|Info SW version = {} release={}| extracts:1.2.34|a.1.1
2023-10-07 12:00:00.02 MDT|1|002|status|info|alphanumeric value|sw_a|Message with alphanumberic value {}| extracts:abc123def
2023-10-07 12:00:00.03 MDT|1|003|status|info|alphanumeric value|sw_a|val:{} flag:{} other:{} on {}| extracts:127.0.0.1:8080|1|x20|X30
2023-10-07 12:00:00.04 MDT|1|004|status|info|alphanumeric value|sw_a|val={} flag = {} other {} on ({})| extracts:4.cd|2|ABC.123_45|30
---------------- PARSED OUTPUT END   ----------------
2023/10/11 18:13:17.527048 main.go:213:    info: total lines with unexpected number of fields=0
```
</font>

Example WITH hashing:
<font size=0.5em>
```
2023/10/11 18:13:05.127175 main.go:77:   debug: user.Current(): &{Uid:501 Gid:20 Username:pauldunn Name:PAUL DUNN HomeDir:/Users/pauldunn}
2023/10/11 18:13:05.127249 main.go:78:    info: Data and logs being saved to directory: /Users/pauldunn/tmp/go-parser
---------------- PARSED OUTPUT START ----------------
2023-10-07 12:00:00.00 MDT|0|0|de0ed065fd3509cd97c9adf2e161328a| extracts:12.Ab.34
2023-10-07 12:00:00.01 MDT|1|001|dce435d44887b230dc76292ff915a541| extracts:1.2.34
2023-10-07 12:00:00.02 MDT|1|002|d4e1f4a37b90a49b218859fc6706256e| extracts:abc123def
2023-10-07 12:00:00.03 MDT|1|003|64d9ff35898788e8d8ba9cbbc1bb5b93| extracts:val:1|other:X30|127.0.0.1:8080
2023-10-07 12:00:00.04 MDT|1|004|233979113ab07438302ab71511c5cfe0| extracts:4.cd|30|2
---------------- PARSED OUTPUT END   ----------------
2023/10/11 18:13:05.128343 main.go:207:    info: len(hashCounts)=5
2023/10/11 18:13:05.128353 main.go:208:   debug: Hashes and counts:
2023/10/11 18:13:05.128361 main.go:210:   debug: hash: de0ed065fd3509cd97c9adf2e161328a, count: 1, value: notification|debug|multi word type|sw_a|Unit {} message (789)
2023/10/11 18:13:05.128370 main.go:210:   debug: hash: dce435d44887b230dc76292ff915a541, count: 1, value: notification|info|SingleWordType|sw_b|Info SW version = {} release=a.1.1
2023/10/11 18:13:05.128377 main.go:210:   debug: hash: d4e1f4a37b90a49b218859fc6706256e, count: 1, value: status|info|alphanumeric value|sw_a|Message with alphanumberic value {} 
2023/10/11 18:13:05.128383 main.go:210:   debug: hash: 64d9ff35898788e8d8ba9cbbc1bb5b93, count: 1, value: status|info|alphanumeric value|sw_a| {} flag:x20 {} on {} 
2023/10/11 18:13:05.128390 main.go:210:   debug: hash: 233979113ab07438302ab71511c5cfe0, count: 1, value: status|info|alphanumeric value|sw_a|val={} flag = {} other {} on (ABC.123_45)
2023/10/11 18:13:05.128397 main.go:213:    info: total lines with unexpected number of fields=0
```
</font>