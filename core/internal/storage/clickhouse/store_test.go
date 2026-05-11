package clickhouse

import (
	"testing"
	"time"
)

func TestClickHouseTimeUsesDateTime64JSONFormat(t *testing.T) {
	value := time.Date(2026, 5, 11, 15, 36, 20, 396022177, time.FixedZone("MSK", 3*60*60))

	got := clickHouseTime(value)
	want := "2026-05-11 12:36:20.396"
	if got != want {
		t.Fatalf("clickHouseTime() = %q, want %q", got, want)
	}
}
