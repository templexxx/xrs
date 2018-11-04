# X-Reed-Solomon

[![GoDoc][1]][2] [![MIT licensed][3]][4] [![Build Status][5]][6] [![Go Report Card][7]][8]

[1]: https://godoc.org/github.com/templexxx/xrs?status.svg
[2]: https://godoc.org/github.com/templexxx/xrs
[3]: https://img.shields.io/badge/license-MIT-blue.svg
[4]: LICENSE
[5]: https://travis-ci.org/templexxx/xrs.svg?branch=master
[6]: https://travis-ci.org/templexxx/xrs
[7]: https://goreportcard.com/badge/github.com/templexxx/xrs
[8]: https://goreportcard.com/report/github.com/templexxx/xrs

## Introduction:
1.  X-Reed-Solomon Erasure Code engine in pure Go.
2.  Fast and Efficient Data Reconstruction in Erasure-code
3.  Saving about 30% I/O in reconstruction
4.  Based on papers: <A “Hitchhiker’s” Guide to Fast and Efﬁcient Data Reconstruction in Erasure-coded Data Centers>
& <A Piggybacking Design Framework for Read-and Download-efﬁcient Distributed Storage Codes>

## Installation
To get the package use the standard:
```bash
go get github.com/templexxx/xrs
```

## Documentation
See the associated [GoDoc](http://godoc.org/github.com/templexxx/xrs)

## Specification
### GOARCH
1. All arch are supported
2. Go1.11(for AVX512)

## Performance

And we must know the benchmark test is quite different with encoding/decoding in practice.

Because in benchmark test loops, the CPU Cache will help a lot. In practice, we must reuse the memory to make the performance become as good as the benchmark test.

Example of performance on my MacBook Pro (Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz)
DataCnt = 10; ParityCnt = 4

### Encoding:

| Vector size | AVX512 (MB/S) | AVX2 (MB/S) |
|-------------|---------------|-------------|
| 4KB         |       --   |    8632     |
| 64KB        |       --   |    7978     |
| 1MB         |      --     |    5967     |

## Links & Deps
* [ReedSolomon](https://github.com/templexxx/reedsolomon)
* [XOR](https://github.com/templexxx/xorsimd)
* [CPU Feature] (https://github.com/templexxx/cpu)

