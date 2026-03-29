package rdb

/**
Unit test for rdb lib
*/

import (
	"testing"

	"github.com/hsuanshao/golibs/ctx"
	"github.com/stretchr/testify/suite"
)

var (
	mockCTX    = ctx.Background()
	mockConfig = Config{
		Master: "postgres://write:postgres@localhost:5432/test_db?sslmode=disable",
		Replicas: []string{
			"postgres://read1:postgres@localhost:5432/test_db?sslmode=disable",
			"postgres://read2:postgres@localhost:5432/test_db?sslmode=disable",
		},
		WriteConnections: 5,
		ReadConnections:  10,
	}
)

type testSuite struct {
	suite.Suite
	dbClient DBClient
}

func TestDBClient(t *testing.T) {
	ctx.SetDebugLevel()
	suite.Run(t, new(testSuite))
}

func (s *testSuite) SetupSuite() {
	s.dbClient, _ = NewRDBClient(mockCTX, &mockConfig)
}

func (s *testSuite) TearDownSuite() {
	s.dbClient.Close()
}

func (s *testSuite) TestExec() {}

func (s *testSuite) TestQuery() {}

func (s *testSuite) TestWithTransaction() {}

func (s *testSuite) TestLeader() {}

func (s *testSuite) TestFollower() {}

func (s *testSuite) TestStats() {}

func (s *testSuite) TestClose() {}

func (s *testSuite) TestNewBatchWriter() {}
