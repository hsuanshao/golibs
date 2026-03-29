package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupDDAgentForTest() {
	/*
		docker run -d --name dd-agent --restart=always -h localdev \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v /proc/:/host/proc/:ro \
		-v /sys/fs/cgroup/:/host/sys/fs/cgroup:ro \
		-p 8125:8125/udp \
		datadog/docker-dd-agent:latest
	*/

	*MetricDest = "datadog"
	*DdHost = "127.0.0.1"
	*DdPort = "8125"
}

func TestDDParseOneTag(t *testing.T) {
	assert.Equal(t, parseTag([]string{"1", "2"}), []string{"1:2"})
}

func TestDDParseTwoTags(t *testing.T) {
	assert.Equal(t, parseTag([]string{"1", "2", "3", "4"}), []string{"1:2", "3:4"})
}

func TestDDParseTagNil(t *testing.T) {
	assert.Equal(t, parseTag(nil), []string(nil))
}

func TestDDParseTagPanic(t *testing.T) {
	assert.Panics(t, func() { parseTag([]string{"1"}) })
}

func TestDDBumpSum(t *testing.T) {
	dm := DDMetrics{}
	setupDDAgentForTest()
	dm.BumpSum("test.key", 1, "k1", "v1")
}

func BenchmarkDDBumpSum1(b *testing.B) {
	dm := DDMetrics{}
	setupDDAgentForTest()
	for n := 0; n < b.N; n++ {
		dm.BumpSum("test.key1", 1, "k1", "v1")
	}
}

func BenchmarkDDBumpSum2(b *testing.B) {
	dm := DDMetrics{}
	setupDDAgentForTest()
	for n := 0; n < b.N; n++ {
		dm.BumpSum("test.key1", 1, "k1", "v1")
		dm.BumpSum("test.key2", 1, "k1", "v1")
	}
}

func BenchmarkDDBumpSumParallel(b *testing.B) {
	dm := DDMetrics{}
	setupDDAgentForTest()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			dm.BumpSum("test.key1", 99, "l1", "v1", "l2", "v2")
			dm.BumpSum("test.key2", 99, "l1", "v1", "l2", "v2")
		}
	})
}

func BenchmarkDDBumpAvg1(b *testing.B) {
	dm := DDMetrics{}
	setupDDAgentForTest()
	for n := 0; n < b.N; n++ {
		dm.BumpAvg("test.key1", 1, "k1", "v1")
	}
}

func BenchmarkDDBumpAvg2(b *testing.B) {
	dm := DDMetrics{}
	setupDDAgentForTest()
	for n := 0; n < b.N; n++ {
		dm.BumpAvg("test.key1", 1, "k1", "v1")
		dm.BumpAvg("test.key2", 1, "k1", "v1")
	}
}

func BenchmarkDDBumpHistogram1(b *testing.B) {
	dm := DDMetrics{}
	setupDDAgentForTest()
	for n := 0; n < b.N; n++ {
		dm.BumpHistogram("test.key1", 1, "k1", "v1")
	}
}

func BenchmarkDDBumpHistogram2(b *testing.B) {
	dm := DDMetrics{}
	setupDDAgentForTest()
	for n := 0; n < b.N; n++ {
		dm.BumpHistogram("test.key1", 1, "k1", "v1")
		dm.BumpHistogram("test.key2", 1, "k1", "v1")
	}
}
