# X-Reed-Solomon (XRS)

[![GoDoc][1]][2] [![MIT licensed][3]][4] [![Build Status][5]][6]

[1]: https://godoc.org/github.com/templexxx/xrs?status.svg
[2]: https://godoc.org/github.com/templexxx/xrs
[3]: https://img.shields.io/badge/license-MIT-blue.svg
[4]: LICENSE
[5]: https://github.com/templexxx/xrs/workflows/unit-test/badge.svg
[6]: https://github.com/templexxx/xrs

## Overview

XRS is a pure-Go erasure coding engine focused on reducing reconstruction I/O while keeping a Reed-Solomon-compatible workflow.

Key points:

- Systematic code with MDS property.
- Optimized reconstruction path with lower read amplification.
- Throughput can exceed 10 GB/s per physical core on suitable hardware.
- Production-proven in a distributed storage system at 10+ PB scale.

This project is based on:

1. [A Hitchhiker's Guide to Fast and Efficient Data Reconstruction in Erasure-coded Data Centers](https://www.cs.cmu.edu/~nihars/publications/Hitchhiker_SIGCOMM14.pdf)
2. [A Piggybacking Design Framework for Read- and Download-efficient Distributed Storage Codes](http://www.cs.cmu.edu/~rvinayak/papers/piggybacking_journal_ieee_tit_2017.pdf)

## Design

XRS splits each vector into two equal halves (`a` and `b`). For example, in a `10+4` layout:

```text
+---------+
| a1 | b1 |
+---------+
| a2 | b2 |
+---------+
| a3 | b3 |
+---------+
    ...
+---------+
|a10 |b10 |
+---------+
|a11 |b11 |
+---------+
|a12 |b12 |
+---------+
|a13 |b13 |
+---------+
```

Because vectors are split into two halves, choose vector sizes that match your disk and I/O characteristics.

The API is intentionally close to a regular Reed-Solomon library, so integration is straightforward.

## Performance

Performance is mainly affected by:

- CPU instruction set extensions.
- Data/parity configuration.
- Vector size.

Benchmark platform:

- MacBook Pro 15-inch (2017)
- Intel Core i7-7700HQ @ 2.80GHz
- Single-core runs

`RS` below refers to [templexxx/reedsolomon](https://github.com/templexxx/reedsolomon).

### Encode

`I/O = (data + parity) * vector_size / cost`

`Base` means no SIMD.

| Data  | Parity  | Vector size | RS I/O (MB/S) | XRS I/O (MB/S) |
|-------|---------|-------------|---------------|----------------|
| 12 | 4 | 4KB | 12658.00 | 10895.15 |
| 12 | 4 | 1MB | 8989.67 | 7530.84 |
| 12 | 4 | 8MB | 8509.06 | 6579.53 |

### Reconstruct

`Need Data` means the amount of data that must be read during reconstruction.

`I/O = (need_data + reconstruct_data_num * vector_size) / cost`

| Data  | Parity  | Vector size | Reconstruct Data Num | RS Need Data | XRS Need Data | RS Cost | XRS Cost | RS I/O (MB/S) | XRS I/O (MB/S) |
|-------|---------|-------------|----------------------|--------------|---------------|---------|----------|---------------|----------------|
| 12 | 4 | 4KB | 1 | 48KB | 34KB | 2140 ns/op | 3567 ns/op | 24885.17 | 10334.99 |
| 12 | 4 | 4KB | 2 | 48KB | 48KB | 3395 ns/op | 5940 ns/op | 16890.41 | 9654.17 |
| 12 | 4 | 4KB | 3 | 48KB | 48KB | 4746 ns/op | 7525 ns/op | 12945.61 | 8164.76 |
| 12 | 4 | 4KB | 4 | 48KB | 48KB | 5958 ns/op | 8851 ns/op | 10999.75 | 7404.41 |

### Update

`I/O = (2 + parity_num + parity_num) * vector_size / cost`

| Data  | Parity  | Vector size | RS I/O (MB/S) | XRS I/O (MB/S) |
|-------|---------|-------------|---------------|----------------|
| 12 | 4 | 4KB | 32739.22 | 26312.14 |

### Replace

`I/O = (parity_num + parity_num + replace_data_num) * vector_size / cost`

| Data  | Parity  | Vector size | Replace Data Num | RS I/O (MB/S) | XRS I/O (MB/S) |
|-------|---------|-------------|------------------|---------------|----------------|
| 12 | 4 | 4KB | 1 | 63908.06 | 44082.57 |
| 12 | 4 | 4KB | 2 | 39966.65 | 26554.30 |
| 12 | 4 | 4KB | 3 | 30007.81 | 19583.16 |
| 12 | 4 | 4KB | 4 | 25138.38 | 16636.82 |
| 12 | 4 | 4KB | 5 | 21261.91 | 14301.15 |
| 12 | 4 | 4KB | 6 | 19833.14 | 13121.98 |
| 12 | 4 | 4KB | 7 | 18395.47 | 12028.10 |
| 12 | 4 | 4KB | 8 | 17364.02 | 11300.55 |

## Notes

Microbenchmarks can differ significantly from production behavior. In benchmark loops, CPU cache effects can materially improve measured throughput.

## Dependencies

- [templexxx/reedsolomon](https://github.com/templexxx/reedsolomon)
- [templexxx/xorsimd](https://github.com/templexxx/xorsimd)
