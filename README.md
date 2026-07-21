# crockford

Go implementation of [Crockford base32](https://www.crockford.com/base32.html) encoding with support for **variable-sized codes**.

Inspired by [earthboundkid/crockford](https://github.com/earthboundkid/crockford) but extended so every function accepts a `size` parameter that controls the exact number of encoded characters produced.

## Installation

```
go get github.com/brysontipton/crockford
```

## Alphabets & encodings

```go
crockford.LowercaseAlphabet  // "0123456789abcdefghjkmnpqrstvwxyz"
crockford.UppercaseAlphabet  // "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

crockford.Upper  // *base32.Encoding — uppercase, no padding
crockford.Lower  // *base32.Encoding — lowercase, no padding
```

## API overview

### Arbitrary bytes

```go
// Encode any byte slice.
s := crockford.Encode(crockford.Upper, data)

// Decode back to bytes (normalises O→0, I/L→1, hyphens removed).
raw, err := crockford.Decode(crockford.Upper, s)
```

### Variable-size random codes

```go
// Generate exactly `size` random Crockford characters.
id := crockford.Random(crockford.Upper, 16)  // e.g. "3XKVJ8BQT4MZFG2N"

// Append variant for zero-allocation pipelines.
buf = crockford.AppendRandom(crockford.Upper, buf, 16)
```

### Variable-size time codes

The encoded strings are **lexicographically sortable** for any chosen size.

| size | bits | max timestamp | resolution |
|-----:|-----:|:-------------|:-----------|
|    8 |   40 | year ~36 812 | 1 s        |
|   10 |   50 | year ~37 M   | 1 s        |
|   12 |   60 | year ~37 B   | 1 s        |

```go
// Encode current time in 8 characters (40-bit Unix seconds).
s := crockford.Time(crockford.Upper, time.Now(), 8)

// 12-character millisecond timestamp.
s := crockford.TimeMillis(crockford.Upper, time.Now(), 12)

// Raw Unix integer.
s := crockford.TimeUnix(crockford.Upper, unixSec, 10)
```

### Variable-size integer codes

```go
// Encode a uint64 in exactly `size` characters (size × 5 bits, big-endian).
// Panics if the value does not fit in size*5 bits.
s := crockford.Uint64(crockford.Upper, 1_000_000, 4)

// Decode back to uint64 (symbol-by-symbol, no byte-boundary rounding).
v, err := crockford.DecodeUint64(crockford.Upper, s)

// Helpers.
crockford.BitLen(40)      // → 8 (minimum chars for 40-bit value)
crockford.MaxValue(8)     // → 0xFFFFFFFFFF (max value in 8 chars)
```

### Hash-based codes

```go
digest := func(b []byte) []byte { h := sha256.Sum256(b); return h[:] }
id := crockford.Hash(crockford.Upper, data, 20, digest)
```

### Normalisation & partitioning

```go
crockford.Normalized("O1l-iI")        // → "011111"  (uppercase, O→0, I/L→1, hyphens stripped)
crockford.Partition("ABCDEFGH", 4)    // → "ABCD-EFGH"
```

## Size guidelines

| use-case | recommended size |
|:---------|:-----------------|
| Short ID (low collision risk) | 8–10 |
| UUID replacement | 26 |
| Timestamp-prefixed ID | 8 (time) + 8–16 (random) |
| Content hash | 16–32 |
