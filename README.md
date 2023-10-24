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
