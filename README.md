# X-Reed-Solomon

[![GoDoc][1]][2] [![MIT licensed][3]][4] [![Build Status][5]][6] [![Go Report Card][7]][8]

[1]: https://godoc.org/github.com/templexxx/xrs?status.svg
[2]: https://godoc.org/github.com/templexxx/xrs
[3]: https://img.shields.io/badge/license-MIT-blue.svg
[4]: LICENSE
[5]: https://github.com/templexxx/xrs/workflows/unit-test/badge.svg
[6]: https://github.com/templexxx/xrs
[7]: https://goreportcard.com/badge/github.com/templexxx/xrs
[8]: https://goreportcard.com/report/github.com/templexxx/xrs

## Introduction:

>- Fast and efficient data reconstruction Erasure Code engine in pure Go.
>
>- [Systematic Codes](https://en.wikipedia.org/wiki/Systematic_code) with [MDS property](https://en.wikipedia.org/wiki/Singleton_bound#MDS_codes).
>
>- [More than 10GB/s per physics core.](https://github.com/templexxx/xrs#performance)
>
>- Saving about 30% I/O in reconstruction.
>
>- Has been used for a distributed storage system with more than 10PB data.
>
>- Based on papers: 
>   1. [<A “Hitchhiker’s” Guide to Fast and Efﬁcient Data Reconstruction in Erasure-coded Data Centers>](https://www.cs.cmu.edu/~nihars/publications/Hitchhiker_SIGCOMM14.pdf)
>   2. [<A Piggybacking Design Framework for Read-and Download-efﬁcient Distributed Storage Codes>](http://www.cs.cmu.edu/~rvinayak/papers/piggybacking_journal_ieee_tit_2017.pdf)

## Getting Started

>-  Make sure you have read the papers.
>
>-  XRS splits row vector into two equal parts.
>
>   e.g. 10+4:
>
    +---------+
	| a1 | b1 |
 	+---------+
 	| a2 | b2 |
 	+---------+
 	| a3 | b3 |
 	+---------+
	    ...
 	+---------+
 	| a10| b10|
 	+---------+
 	| a11| b11|
 	+---------+
 	| a12| b12|
 	+---------+
	| a13| b13|
 	+---------+
 	
>>- So it's important to choose a fit size for reading/write disks efficiently.
>
>- APIs are almost as same as normal Reed-Solomon Erasure Codes.

## Performance

Performance depends mainly on:

>- CPU instruction extension.
>
>- Number of data/parity row vectors.

**Platform:** 
 
*MacBook Pro 15-inch, 2017 (Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz)*
 
>All test run on a single Core.
>
>RS means Reed-Solomon Codes(for comparing), the RS lib is [here](https://github.com/templexxx/reedsolomon)

### Encode:

`I/O = (data + parity) * vector_size / cost`

*Base means no SIMD.*

| Data  | Parity  | Vector size | RS I/O (MB/S) |  XRS I/O (MB/S) |
|-------|---------|-------------|-------------|---------------|
|12|4|4KB|    12658.00     |    10895.15      | 
|12|4|1MB|      8989.67   |   7530.84       |   
|12|4|8MB|     8509.06    |    6579.53      |   

### Reconstruct:

`Need Data = Data size need read in reconstruction`

`I/O = (need_data + reconstruct_data_num * vector_size) / cost`

| Data  | Parity  | Vector size | Reconstruct Data Num |   RS Need Data |  XRS Need Data | RS Cost | XRS Cost | RS I/O (MB/S) |  XRS I/O (MB/S) |
|-------|---------|-------------|-------------|---------------|---------------|-------------|---------------|-------------|---------------|
|12|4|4KB| 1         |   48KB    |   34KB    |    2140 ns/op   |   3567 ns/op    |    24885.17  |10334.99|
|12|4|4KB| 2        |     48KB   |    48KB     |   3395 ns/op    |   5940 ns/op    |    16890.41   |9654.17|
|12|4|4KB| 3         |     48KB     |   48KB     |   4746 ns/op    |   7525 ns/op    |  12945.61     |8164.76|
|12|4|4KB| 4         |     48KB     |   48KB     |    5958 ns/op   |    8851 ns/op   |   10999.75    |7404.41|

### Update:

`I/O = (2 + parity_num + parity_num) * vector_size / cost`

| Data  | Parity  | Vector size | RS I/O (MB/S) | XRS I/O (MB/S) |
|-------|---------|-------------|-------------|-------------|
|12|4|4KB|     32739.22    |      26312.14  |

### Replace:

`I/O = (parity_num + parity_num + replace_data_num) * vector_size / cost`

| Data  | Parity  | Vector size | Replace Data Num |  RS I/O (MB/S) |XRS I/O (MB/S) |
|-------|---------|-------------|-------------|---------------|-------------|
|12|4|4KB| 1         |     63908.06     |   44082.57        | 
|12|4|4KB| 2        |   39966.65   |         26554.30   | 
|12|4|4KB| 3         |    30007.81      |    19583.16       | 
|12|4|4KB| 4         |    25138.38       |    16636.82         |  
|12|4|4KB| 5         |    21261.91      |     14301.15      | 
|12|4|4KB| 6         |     19833.14     |    13121.98       | 
|12|4|4KB| 7         |    18395.47    |    12028.10       | 
|12|4|4KB| 8         |     17364.02     |   11300.55        | 

**PS:**

*And we must know the benchmark test is quite different with encoding/decoding in practice.
Because in benchmark test loops, the CPU Cache may help a lot.*

## Links & Deps
* [Reed-Solomon](https://github.com/templexxx/reedsolomon)
* [XOR](https://github.com/templexxx/xorsimd)
