package metrics

import (
	"testing"
)

func TestMetricBumpSum(t *testing.T) {
	m := New("test")
	m.BumpSum("metric", 1, "key1", "val1")
	m.BumpSum("metric", 2, "key1", "val2")
}

func TestMetricBumpSumWithPanic(t *testing.T) {
	m := New("test")
	m.BumpSum("metric", 1, "key1", "val1")
	m.BumpSum("metric", 2, "key1", "val2", "key2", "val2")
}

func TestMetricBumpTimeWithPanic(t *testing.T) {
	m := New("test")
	t1 := m.BumpTime("metric", "key1", "val1")
	t1.End()
	t2 := m.BumpTime("metric", "key1", "val2", "key2", "val2")
	t2.End()
}
