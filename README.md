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
debug: user.Current(): &{Uid:501 Gid:20 Username:pauldunn Name:PAUL DUNN HomeDir:/Users/pauldunn}
info: Data and logs being saved to directory: /Users/pauldunn/tmp/go-parser
info: 2023-10-07 12:00:00.00 MDT|0|0|notification|debug|multi word type|sw_a|Unit {} message| extracts:12.Ab.34
info: 2023-10-07 12:00:00.01 MDT|1|001|notification|info|SingleWordType|sw_b|Info SW version = {} message| extracts:1.2.34
info: 2023-10-07 12:00:00.02 MDT|1|002|status|info|alphanumeric value|sw_a|Message with alphanumberic value {} | extracts:abc123def
info: 2023-10-07 12:00:00.03 MDT|1|003|status|info|alphanumeric value|sw_a|val={} flag = {} other {} | extracts:3.AB|20|1
info: 2023-10-07 12:00:00.03 MDT|1|003|status|info|alphanumeric value|sw_a|val={} flag = {} other {} | extracts:4.cd|30|2
info: total lines with unexpected number of fields=0
```
</font>

Example WITH hashing:
<font size=0.5em>
```
go run ./main.go -datafile="./parser/test/test_extract.txt" -inputfile="exampleInputWithHashing.json" -loglevel=0
debug: user.Current(): &{Uid:501 Gid:20 Username:pauldunn Name:PAUL DUNN HomeDir:/Users/pauldunn}
info: Data and logs being saved to directory: /Users/pauldunn/tmp/go-parser
info: 2023-10-07 12:00:00.00 MDT|0|0|d4f074fb52bfb9641536b6af3f28309f| extracts:12.Ab.34
info: 2023-10-07 12:00:00.01 MDT|1|001|fa3396c23df603e6f20cd904fb88a5e7| extracts:1.2.34
info: 2023-10-07 12:00:00.02 MDT|1|002|d4e1f4a37b90a49b218859fc6706256e| extracts:abc123def
info: 2023-10-07 12:00:00.03 MDT|1|003|aa3294bc37a046356dd9d1cb89152583| extracts:3.AB|20|1
info: 2023-10-07 12:00:00.03 MDT|1|003|aa3294bc37a046356dd9d1cb89152583| extracts:4.cd|30|2
info: len(hashCounts)=4
debug: Hashes and counts:
debug: hash: aa3294bc37a046356dd9d1cb89152583, count: 2, value: status|info|alphanumeric value|sw_a|val={} flag = {} other {} 
debug: hash: d4f074fb52bfb9641536b6af3f28309f, count: 1, value: notification|debug|multi word type|sw_a|Unit {} message
debug: hash: fa3396c23df603e6f20cd904fb88a5e7, count: 1, value: notification|info|SingleWordType|sw_b|Info SW version = {} message
debug: hash: d4e1f4a37b90a49b218859fc6706256e, count: 1, value: status|info|alphanumeric value|sw_a|Message with alphanumberic value {} 
info: total lines with unexpected number of fields=0
```
</font>