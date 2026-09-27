package errorx_test

import (
	"errors"
	"testing"
)

var sink any

func TestSentinelZeroAlloc(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		sink = error(ErrOrderFail)
	})
	if allocs != 0 {
		t.Fatalf("sentinel allocs = %v, want 0", allocs)
	}
}

func BenchmarkCreateChain(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sink = ErrUserNotFound.Str("id", "u1").Int("age", 18).Wrap(errors.New("x"))
	}
}

func BenchmarkErrorString(b *testing.B) {
	err := ErrUserNotFound.Str("id", "u1")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sink = err.Error()
	}
}

func BenchmarkStackCapture(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sink = ErrOrderFail.Stack()
	}
}

func BenchmarkIs(b *testing.B) {
	err := ErrUserNotFound.Str("id", "u1").Wrap(errors.New("x"))
	for i := 0; i < b.N; i++ {
		sink = errors.Is(err, ErrUserNotFound)
	}
}
