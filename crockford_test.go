package crockford_test

import (
	"crypto/sha256"
	"strings"
	"testing"
	"time"

	"github.com/brysontipton/crockford"
)

// ---------------------------------------------------------------------------
// EncodedLen / DecodedLen
// ---------------------------------------------------------------------------

func TestEncodedLen(t *testing.T) {
	cases := []struct{ src, want int }{
		{0, 0},
		{1, 2},  // 8 bits → 2 chars (10 bits / 5)
		{5, 8},  // 40 bits → 8 chars
		{6, 10}, // 48 bits → 10 chars
		{16, 26},
	}
	for _, c := range cases {
		if got := crockford.EncodedLen(c.src); got != c.want {
			t.Errorf("EncodedLen(%d) = %d, want %d", c.src, got, c.want)
		}
	}
}

func TestDecodedLen(t *testing.T) {
	cases := []struct{ enc, want int }{
		{0, 0},
		{2, 1},
		{8, 5},
		{10, 6},
		{26, 16},
	}
	for _, c := range cases {
		if got := crockford.DecodedLen(c.enc); got != c.want {
			t.Errorf("DecodedLen(%d) = %d, want %d", c.enc, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Encode / Decode round-trip
// ---------------------------------------------------------------------------

func TestEncodeDecode(t *testing.T) {
	inputs := [][]byte{
		{},
		{0},
		{0xFF},
		{0xDE, 0xAD, 0xBE, 0xEF},
		[]byte("Hello, Crockford!"),
	}
	for _, input := range inputs {
		if len(input) == 0 {
			continue // base32 of empty is empty; skip trivial case
		}
		enc := crockford.Encode(crockford.Upper, input)
		dec, err := crockford.Decode(crockford.Upper, enc)
		if err != nil {
			t.Errorf("Decode(%q) error: %v", enc, err)
			continue
		}
		if string(dec) != string(input) {
			t.Errorf("round-trip mismatch for %x: got %x", input, dec)
		}
	}
}

// ---------------------------------------------------------------------------
// Random
// ---------------------------------------------------------------------------

func TestRandomLength(t *testing.T) {
	for _, size := range []int{1, 5, 8, 10, 13, 16, 20, 32, 64} {
		got := crockford.Random(crockford.Upper, size)
		if len(got) != size {
			t.Errorf("Random(Upper, %d) len = %d, want %d", size, len(got), size)
		}
	}
}

func TestRandomAlphabet(t *testing.T) {
	alphabet := crockford.UppercaseAlphabet
	s := crockford.Random(crockford.Upper, 100)
	for i, c := range s {
		if !strings.ContainsRune(alphabet, c) {
			t.Errorf("Random output char %d (%c) not in alphabet", i, c)
		}
	}
}

func TestRandomUniqueness(t *testing.T) {
	a := crockford.Random(crockford.Upper, 16)
	b := crockford.Random(crockford.Upper, 16)
	if a == b {
		t.Errorf("two Random(16) calls returned the same value %q", a)
	}
}

func TestRandomLowercase(t *testing.T) {
	alphabet := crockford.LowercaseAlphabet
	s := crockford.Random(crockford.Lower, 20)
	for i, c := range s {
		if !strings.ContainsRune(alphabet, c) {
			t.Errorf("Random(Lower) char %d (%c) not in lowercase alphabet", i, c)
		}
	}
}

// ---------------------------------------------------------------------------
// Time
// ---------------------------------------------------------------------------

func TestTimeLength(t *testing.T) {
	now := time.Now()
	for _, size := range []int{1, 4, 8, 10, 12, 13} {
		got := crockford.Time(crockford.Upper, now, size)
		if len(got) != size {
			t.Errorf("Time(Upper, now, %d) len = %d, want %d", size, len(got), size)
		}
	}
}

func TestTimeSortable(t *testing.T) {
	t1 := time.Unix(1_000_000_000, 0)
	t2 := time.Unix(2_000_000_000, 0)
	s1 := crockford.Time(crockford.Upper, t1, 8)
	s2 := crockford.Time(crockford.Upper, t2, 8)
	if s1 >= s2 {
		t.Errorf("Time encoding not sortable: %q >= %q", s1, s2)
	}
}

func TestTimeMillisLength(t *testing.T) {
	now := time.Now()
	for _, size := range []int{8, 10, 12, 13} {
		got := crockford.TimeMillis(crockford.Upper, now, size)
		if len(got) != size {
			t.Errorf("TimeMillis(Upper, now, %d) len = %d, want %d", size, len(got), size)
		}
	}
}

func TestTimeMillisSortable(t *testing.T) {
	t1 := time.UnixMilli(1_000_000_000_000)
	t2 := time.UnixMilli(2_000_000_000_000)
	s1 := crockford.TimeMillis(crockford.Upper, t1, 12)
	s2 := crockford.TimeMillis(crockford.Upper, t2, 12)
	if s1 >= s2 {
		t.Errorf("TimeMillis encoding not sortable: %q >= %q", s1, s2)
	}
}

// ---------------------------------------------------------------------------
// Uint64
// ---------------------------------------------------------------------------

func TestUint64RoundTrip(t *testing.T) {
	cases := []struct {
		v    uint64
		size int
	}{
		{0, 1},
		{31, 1},
		{1024, 4},
		{0xDEADBEEF, 8},
		{crockford.MaxValue(12), 12},
	}
	for _, c := range cases {
		enc := crockford.Uint64(crockford.Upper, c.v, c.size)
		if len(enc) != c.size {
			t.Errorf("Uint64(%d, %d) len = %d, want %d", c.v, c.size, len(enc), c.size)
		}
		dec, err := crockford.DecodeUint64(crockford.Upper, enc)
		if err != nil {
			t.Errorf("DecodeUint64(%q) error: %v", enc, err)
			continue
		}
		// Re-mask to the bit width so we can compare.
		bitWidth := c.size * 5
		var mask uint64
		if bitWidth >= 64 {
			mask = ^uint64(0)
		} else {
			mask = (1 << uint(bitWidth)) - 1
		}
		want := c.v & mask
		if dec != want {
			t.Errorf("Uint64 round-trip for %d (size %d): got %d, want %d", c.v, c.size, dec, want)
		}
	}
}

func TestUint64Sortable(t *testing.T) {
	for _, size := range []int{4, 8, 10, 13} {
		a := crockford.Uint64(crockford.Upper, 100, size)
		b := crockford.Uint64(crockford.Upper, 200, size)
		if a >= b {
			t.Errorf("Uint64 size %d not sortable: %q >= %q", size, a, b)
		}
	}
}

func TestBitLen(t *testing.T) {
	cases := []struct{ bits, want int }{
		{5, 1},
		{40, 8},
		{41, 9},
		{50, 10},
	}
	for _, c := range cases {
		if got := crockford.BitLen(c.bits); got != c.want {
			t.Errorf("BitLen(%d) = %d, want %d", c.bits, got, c.want)
		}
	}
}

func TestMaxValue(t *testing.T) {
	if crockford.MaxValue(1) != 31 {
		t.Errorf("MaxValue(1) = %d, want 31", crockford.MaxValue(1))
	}
	if crockford.MaxValue(8) != 0xFFFFFFFFFF {
		t.Errorf("MaxValue(8) = %d, want %d", crockford.MaxValue(8), uint64(0xFFFFFFFFFF))
	}
}

// ---------------------------------------------------------------------------
// Normalisation
// ---------------------------------------------------------------------------

func TestNormalized(t *testing.T) {
	cases := []struct{ in, want string }{
		{"0OoIiLl", "0001111"},
		{"ABC-DEF", "ABCDEF"},        // hyphens removed
		{"abcdefg", "ABCDEFG"},       // lowercase → upper
		{"ABCDEFGHJK", "ABCDEFGHJK"}, // already normalised
	}
	for _, c := range cases {
		got := crockford.Normalized(c.in)
		if got != c.want {
			t.Errorf("Normalized(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Partition
// ---------------------------------------------------------------------------

func TestPartition(t *testing.T) {
	cases := []struct {
		in, want string
		gap      int
	}{
		{"ABCDEFGH", "ABCD-EFGH", 4},
		{"ABCDEFGHI", "ABC-DEF-GHI", 3},
		{"AB", "AB", 4},
		{"ABCDE", "ABCDE", 5},
		{"ABCDEF", "ABCDE-F", 5},
	}
	for _, c := range cases {
		got := crockford.Partition(c.in, c.gap)
		if got != c.want {
			t.Errorf("Partition(%q, %d) = %q, want %q", c.in, c.gap, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Hash
// ---------------------------------------------------------------------------

func TestHash(t *testing.T) {
	digest := func(b []byte) []byte { h := sha256.Sum256(b); return h[:] }
	data := []byte("hello world")

	for _, size := range []int{4, 8, 10, 16, 20, 26, 32, 52} {
		got := crockford.Hash(crockford.Upper, data, size, digest)
		if len(got) != size {
			t.Errorf("Hash(size=%d) len = %d, want %d", size, len(got), size)
		}
	}

	// Same input → same output (deterministic).
	a := crockford.Hash(crockford.Upper, data, 20, digest)
	b := crockford.Hash(crockford.Upper, data, 20, digest)
	if a != b {
		t.Errorf("Hash is not deterministic: %q != %q", a, b)
	}
}

// ---------------------------------------------------------------------------
// Append variants (smoke test)
// ---------------------------------------------------------------------------

func TestAppendVariants(t *testing.T) {
	base := []byte("PREFIX-")
	result := crockford.AppendRandom(crockford.Upper, base, 8)
	if !strings.HasPrefix(string(result), "PREFIX-") {
		t.Errorf("AppendRandom did not preserve prefix")
	}
	if len(result) != len(base)+8 {
		t.Errorf("AppendRandom len = %d, want %d", len(result), len(base)+8)
	}
}
