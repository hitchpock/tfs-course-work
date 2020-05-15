package robot

import (
	"database/sql"
	"testing"
	"time"
)

func (t *NullTime) ScanCustom(value interface{}) error {
	ni := sql.NullTime(*t)
	return ni.Scan(value)
}

func BenchmarkScanCustom(b *testing.B) {
	t := NullTime{}
	value := time.Time{}

	for i := 0; i < b.N; i++ {
		_ = t.ScanCustom(value)
	}
}

func BenchmarkScanPQ(b *testing.B) {
	t := NullTime{}
	value := time.Time{}

	for i := 0; i < b.N; i++ {
		_ = t.Scan(value)
	}
}
