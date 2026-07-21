// Package crockford implements the Crockford base32 encoding with support
// for variable-sized codes.
//
// See https://www.crockford.com/base32.html
package crockford

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"slices"
	"time"
)

// Alphabet constants for Crockford base32.
const (
	LowercaseAlphabet = "0123456789abcdefghjkmnpqrstvwxyz"
	UppercaseAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
)

// Pre-built encodings for convenience.
var (
	Lower = base32.NewEncoding(LowercaseAlphabet).WithPadding(base32.NoPadding)
	Upper = base32.NewEncoding(UppercaseAlphabet).WithPadding(base32.NoPadding)
)

// EncodedLen returns the number of Crockford base32 characters needed to
// encode srcLen raw bytes.
func EncodedLen(srcLen int) int {
	if srcLen == 0 {
		return 0
	}
	// Each 5 bits → 1 character; ceil(srcLen*8 / 5)
	return (srcLen*8 + 4) / 5
}

// DecodedLen returns the maximum number of raw bytes that encodedLen
// Crockford base32 characters can decode into.
func DecodedLen(encodedLen int) int {
	if encodedLen == 0 {
		return 0
	}
	// Each character carries 5 bits; floor(encodedLen*5 / 8)
	return encodedLen * 5 / 8
}

// rawBytesForChars returns the minimum number of raw bytes whose encoding
// produces at least n characters.
func rawBytesForChars(n int) int {
	// We need ceil(n * 5 / 8) bytes.
	return (n*5 + 7) / 8
}

// AppendEncode appends the Crockford base32 encoding of src onto dst.
func AppendEncode(e *base32.Encoding, dst, src []byte) []byte {
	return e.AppendEncode(dst, src)
}

// Encode returns the Crockford base32 encoding of src.
func Encode(e *base32.Encoding, src []byte) string {
	return string(AppendEncode(e, nil, src))
}

// AppendDecode appends the decoding of the Crockford base32-encoded src onto
// dst. Normalisation (O→0, I/L→1, lowercase, hyphens) is applied first.
// Returns an error if src contains invalid characters after normalisation.
func AppendDecode(e *base32.Encoding, dst, src []byte) ([]byte, error) {
	norm := AppendNormalized(nil, src)
	decoded, err := e.AppendDecode(dst, norm)
	if err != nil {
		return dst, fmt.Errorf("crockford: decode error: %w", err)
	}
	return decoded, nil
}

// Decode decodes the Crockford base32-encoded string s.
func Decode(e *base32.Encoding, s string) ([]byte, error) {
	return AppendDecode(e, nil, []byte(s))
}

// ---------------------------------------------------------------------------
// Variable-size random codes
// ---------------------------------------------------------------------------

// AppendRandom appends size Crockford base32-encoded characters of
// cryptographically random data onto dst and returns the resulting slice.
// size must be >= 1.
//
// Because Crockford base32 encodes 5 bits per character, not every size maps
// to a whole number of bytes. The implementation generates enough random bytes,
// encodes them, and returns exactly size characters (high bits of the last
// partial byte are zero).
func AppendRandom(e *base32.Encoding, dst []byte, size int) []byte {
	if size < 1 {
		panic("crockford: size must be >= 1")
	}
	rawLen := rawBytesForChars(size)
	raw := make([]byte, rawLen)
	if _, err := rand.Read(raw); err != nil {
		panic(err)
	}
	encoded := e.AppendEncode(nil, raw)
	return append(dst, encoded[:size]...)
}

// Random returns size Crockford base32-encoded characters of
// cryptographically random data. size must be >= 1.
func Random(e *base32.Encoding, size int) string {
	return string(AppendRandom(e, nil, size))
}

// ---------------------------------------------------------------------------
// Variable-size time codes
// ---------------------------------------------------------------------------

// AppendTime appends size Crockford base32-encoded characters representing
// the Unix timestamp of t onto dst and returns the resulting slice.
//
// size controls the precision and range of the encoded time:
//   - size == 8  → 40-bit Unix seconds  (~year 36 812, resolution 1 s)
//   - size == 10 → 50-bit Unix seconds  (~year 37 million)
//   - size == 12 → 60-bit Unix seconds  (practical upper limit)
//   - size == 10 with millisecond timestamps, etc.
//
// The encoded integer is the Unix time in seconds, stored big-endian in the
// minimum number of bytes that the requested size requires. The resulting
// string is lexicographically sortable as long as the timestamp fits in the
// bit width (bitWidth = size * 5).
//
// size must be between 1 and 13 inclusive.
func AppendTime(e *base32.Encoding, dst []byte, t time.Time, size int) []byte {
	return AppendTimeUnix(e, dst, t.Unix(), size)
}

// Time returns size Crockford base32-encoded characters for the Unix
// timestamp of t. See AppendTime for details.
func Time(e *base32.Encoding, t time.Time, size int) string {
	return string(AppendTime(e, nil, t, size))
}

// AppendTimeUnix appends size Crockford base32-encoded characters for the
// given Unix timestamp (seconds) onto dst. See AppendTime for details.
func AppendTimeUnix(e *base32.Encoding, dst []byte, unix int64, size int) []byte {
	if size < 1 || size > 13 {
		panic("crockford: time size must be between 1 and 13")
	}
	// Encode the full 64-bit timestamp big-endian, then take the first `size`
	// characters. This preserves the most-significant bits so the result is
	// lexicographically sortable for any size. No overflow check: small sizes
	// intentionally represent a coarser resolution of the timestamp.
	rawLen := rawBytesForChars(size)
	if rawLen > 8 {
		rawLen = 8
	}
	u := uint64(unix)
	var buf [8]byte
	buf[0] = byte(u >> 56)
	buf[1] = byte(u >> 48)
	buf[2] = byte(u >> 40)
	buf[3] = byte(u >> 32)
	buf[4] = byte(u >> 24)
	buf[5] = byte(u >> 16)
	buf[6] = byte(u >> 8)
	buf[7] = byte(u)
	encoded := e.AppendEncode(nil, buf[:rawLen])
	return append(dst, encoded[:size]...)
}

// TimeUnix returns size Crockford base32-encoded characters for the given
// Unix timestamp (seconds). See AppendTime for details.
func TimeUnix(e *base32.Encoding, unix int64, size int) string {
	return string(AppendTimeUnix(e, nil, unix, size))
}

// AppendTimeMillis appends size Crockford base32-encoded characters for the
// Unix timestamp of t in milliseconds onto dst. Millisecond precision gives
// ~35-bit range per additional character compared to second precision.
//
// size must be between 1 and 13 inclusive.
func AppendTimeMillis(e *base32.Encoding, dst []byte, t time.Time, size int) []byte {
	ms := t.UnixMilli()
	return AppendTimeUnix(e, dst, ms, size)
}

// TimeMillis returns size Crockford base32-encoded characters for the Unix
// timestamp of t in milliseconds.
func TimeMillis(e *base32.Encoding, t time.Time, size int) string {
	return string(AppendTimeMillis(e, nil, t, size))
}

// ---------------------------------------------------------------------------
// Encode arbitrary integer values
// ---------------------------------------------------------------------------

// AppendUint64 appends the Crockford base32 encoding of v using exactly size
// characters onto dst. The value is stored big-endian. Panics if v does not
// fit in size*5 bits.
func AppendUint64(e *base32.Encoding, dst []byte, v uint64, size int) []byte {
	if size < 1 || size > 13 {
		panic("crockford: size must be between 1 and 13")
	}
	bitWidth := size * 5
	if bitWidth < 64 && v >= (1<<uint(bitWidth)) {
		panic(fmt.Sprintf("crockford: value %d does not fit in %d bits (%d chars)", v, bitWidth, size))
	}
	// Left-align the value in rawLen bytes so the base32 encoding reads the
	// correct bits from MSB to LSB. shift = rawLen*8 - bitWidth.
	// For size=13, rawLen=8 and bitWidth=65; we cap shift at 0 since uint64
	// only provides 64 bits (the last encoded character carries 1 padding bit).
	rawLen := rawBytesForChars(size)
	if rawLen > 8 {
		rawLen = 8
	}
	shift := rawLen*8 - bitWidth
	if shift < 0 {
		shift = 0
	}
	aligned := v << uint(shift) // shift is always 0..7, safe for uint64
	var buf [8]byte
	buf[0] = byte(aligned >> 56)
	buf[1] = byte(aligned >> 48)
	buf[2] = byte(aligned >> 40)
	buf[3] = byte(aligned >> 32)
	buf[4] = byte(aligned >> 24)
	buf[5] = byte(aligned >> 16)
	buf[6] = byte(aligned >> 8)
	buf[7] = byte(aligned)
	encoded := e.AppendEncode(nil, buf[8-rawLen:])
	return append(dst, encoded[:size]...)
}

// Uint64 returns the Crockford base32 encoding of v in exactly size
// characters.
func Uint64(e *base32.Encoding, v uint64, size int) string {
	return string(AppendUint64(e, nil, v, size))
}

// DecodeUint64 decodes a variable-length Crockford base32-encoded string into
// a uint64. Each character contributes 5 bits (big-endian). Returns an error
// if the string is longer than 13 characters or contains invalid characters.
func DecodeUint64(e *base32.Encoding, s string) (uint64, error) {
	if len(s) > 13 {
		return 0, fmt.Errorf("crockford: input too long for uint64 (%d chars)", len(s))
	}
	norm := AppendNormalized(nil, []byte(s))
	// Decode symbol-by-symbol using a lookup table built from the alphabet.
	table := buildDecodeTable(e)
	var v uint64
	for _, c := range norm {
		sym, ok := table[rune(c)]
		if !ok {
			return 0, fmt.Errorf("crockford: invalid character %q", c)
		}
		v = (v << 5) | uint64(sym)
	}
	return v, nil
}

// buildDecodeTable returns a map from (uppercase) rune to its 5-bit value
// for the given encoding.
func buildDecodeTable(e *base32.Encoding) map[rune]byte {
	// Encode each possible 5-bit value (0-31) as a single symbol by encoding
	// a byte whose top 5 bits equal the value.
	table := make(map[rune]byte, 32)
	for i := 0; i < 32; i++ {
		// Shift value to top 5 bits of a byte, encode 1 byte → 2 chars.
		encoded := e.AppendEncode(nil, []byte{byte(i << 3)})
		if len(encoded) >= 1 {
			table[rune(encoded[0])] = byte(i)
		}
	}
	return table
}

// BitLen returns the minimum number of Crockford base32 characters needed to
// represent any value with the given number of significant bits.
// For example, BitLen(40) == 8.
func BitLen(bitWidth int) int {
	return (bitWidth + 4) / 5
}

// MaxValue returns the maximum uint64 that fits in size Crockford characters
// (i.e., in size*5 bits). For size >= 13 this saturates at ^uint64(0).
func MaxValue(size int) uint64 {
	bits_ := size * 5
	if bits_ >= 64 {
		return ^uint64(0)
	}
	return (1 << uint(bits_)) - 1
}

// ---------------------------------------------------------------------------
// Normalisation
// ---------------------------------------------------------------------------

func normUpper(c byte) byte {
	switch {
	case c == '0' || c == 'O' || c == 'o':
		return '0'
	case c == '1' || c == 'I' || c == 'i' || c == 'L' || c == 'l':
		return '1'
	case (c >= '2' && c <= '9') ||
		(c >= 'A' && c <= 'H') ||
		(c >= 'J' && c <= 'N') ||
		(c >= 'P' && c <= 'T') ||
		(c >= 'V' && c <= 'Z') ||
		c == '*' || c == '~' || c == '$' || c == '=' || c == 'U':
		return c
	case (c >= 'a' && c <= 'h') ||
		(c >= 'j' && c <= 'n') ||
		(c >= 'p' && c <= 't') ||
		(c >= 'v' && c <= 'z') ||
		c == 'u':
		return c - ('a' - 'A')
	}
	return 0
}

// AppendNormalized appends onto dst a normalised version of src:
//   - O/o → 0
//   - I/i/L/l → 1
//   - lowercase → uppercase
//   - invalid characters (e.g. hyphens) are removed
func AppendNormalized(dst, src []byte) []byte {
	dst = slices.Grow(dst, len(src))
	for _, c := range src {
		if r := normUpper(c); r != 0 {
			dst = append(dst, r)
		}
	}
	return dst
}

// Normalized returns a normalised version of the Crockford encoded string s.
func Normalized(s string) string {
	return string(AppendNormalized(nil, []byte(s)))
}

// ---------------------------------------------------------------------------
// Partitioning (readability hyphens)
// ---------------------------------------------------------------------------

// AppendPartition appends onto dst the result of partitioning src with hyphens
// every gap characters. The last group may be shorter.
// gap must be >= 1. This function is not Unicode-aware.
func AppendPartition(dst, src []byte, gap int) []byte {
	if gap < 1 {
		panic("crockford: invalid gap")
	}
	if len(src) == 0 {
		return dst
	}
	gaps := len(src) / gap
	rem := len(src) % gap
	if rem == 0 && gaps > 0 {
		gaps--
	}
	n := gaps + len(src)
	dst = slices.Grow(dst, n)
	start := len(dst)
	dst = dst[:start+n]
	r := dst[start:]

	for {
		var tailDst, tailSrc []byte
		tailLen := gap
		if rem != 0 {
			tailLen = rem
			rem = 0
		}
		src, tailSrc = splitLast(src, tailLen)
		r, tailDst = splitLast(r, tailLen)
		copy(tailDst, tailSrc)
		if gaps < 1 {
			return dst
		}
		r, tailDst = splitLast(r, 1)
		tailDst[0] = '-'
		gaps--
	}
}

// Partition returns s with hyphens inserted every gap characters.
func Partition(s string, gap int) string {
	return string(AppendPartition(nil, []byte(s), gap))
}

// ---------------------------------------------------------------------------
// Hash-based codes (variable size)
// ---------------------------------------------------------------------------

// AppendHash appends size Crockford base32-encoded characters derived from
// hashing src using the provided hash function (passed as a function that
// returns digest bytes). This is useful for building content-addressed IDs of
// arbitrary length.
//
//	digest := func(b []byte) []byte { h := sha256.Sum256(b); return h[:] }
//	id := crockford.AppendHash(crockford.Upper, nil, data, 16, digest)
func AppendHash(e *base32.Encoding, dst, src []byte, size int, hash func([]byte) []byte) []byte {
	if size < 1 {
		panic("crockford: size must be >= 1")
	}
	digest := hash(src)
	encoded := e.AppendEncode(nil, digest)
	if size > len(encoded) {
		panic(fmt.Sprintf("crockford: requested size %d exceeds hash output of %d chars", size, len(encoded)))
	}
	return append(dst, encoded[:size]...)
}

// Hash returns size Crockford base32-encoded characters derived from hashing
// src using the provided hash function.
func Hash(e *base32.Encoding, src []byte, size int, hash func([]byte) []byte) string {
	return string(AppendHash(e, nil, src, size, hash))
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func splitLast(b []byte, n int) ([]byte, []byte) {
	return b[:len(b)-n], b[len(b)-n:]
}
