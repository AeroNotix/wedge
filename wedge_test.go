package wedge

import (
	"runtime"
	"testing"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func BenchmarkLockMap(b *testing.B) {
	b.StopTimer()
	m := NewLockMap()
	b.StartTimer()
	for x := 0; x < b.N; x++ {
		m.Insert(x, x)
	}
}

func BenchmarkChanMap(b *testing.B) {
	b.StopTimer()
	m := NewSafeMap()
	b.StartTimer()
	for x := 0; x < b.N; x++ {
		m.Insert(x, x)
	}
}

func BenchmarkLockMapFind(b *testing.B) {
	b.StopTimer()
	m := NewLockMap()
	for x := 0; x < b.N; x++ {
		m.Insert(x, x)
	}
	b.StartTimer()
	for x := 0; x < b.N; x++ {
		m.Find(x)
	}
}

func BenchmarkChanMapFind(b *testing.B) {
	b.StopTimer()
	m := NewSafeMap()
	for x := 0; x < b.N; x++ {
		m.Insert(x, x)
	}
	b.StartTimer()
	for x := 0; x < b.N; x++ {
		m.Find(x)
	}
}
