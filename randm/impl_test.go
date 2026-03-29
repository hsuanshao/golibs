package randm

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

var (
	rTime, _ = time.Parse("2006-01-02T15:04:05-07:00", "2025-01-15T01:06:08+00:00")
	fTime, _ = time.Parse("2006-01-02T15:04:05-07:00", "2028-01-01T00:00:00+00:00")
)

func TestSuite(t *testing.T) {
	suite.Run(t, new(randSuite))
}

type randSuite struct {
	suite.Suite
	node    int64
	node2   int64
	m       Method
	m2      Method
	futureM Method
}

func (s *randSuite) SetupSuite() {
	s.node = 1
	s.m = NewDice(s.node, rTime)
	s.node2 = 20
	s.m2 = NewDice(s.node2, rTime)
	s.futureM = NewDice(s.node, fTime)
}

func (s *randSuite) TestGenerateRID() {
	//	rand.Seed(int64(10))
	testcase := []struct {
		Case         string
		MockFunc     func()
		ExecuteTimes uint
	}{
		{
			Case:         "Node 1",
			ExecuteTimes: 1000,
		},
	}

	//var td time.Duration
	for idx, c := range testcase {
		rids := make([]RID, c.ExecuteTimes)

		for i := 0; i < int(c.ExecuteTimes); i++ {
			// st := rand.Intn(9)

			// td = time.Duration(st) * time.Millisecond
			// time.Sleep(td)

			rids[i] = s.m.GenerateRID()
		}

		// condition 1 check
		for i := 0; i < int(c.ExecuteTimes)-1; i++ {
			if !(rids[i] < rids[i+1] && rids[i] != rids[i+1]) {
				s.Error(fmt.Errorf("case %d: i RID %v, i+1 RID %v, doesn't match RID definition", idx, rids[i], rids[i+1]))
			}
		}
	}
}

func (s *randSuite) TestGetNodeFromRID() {
	// Generate a RID from node 1 (s.m)
	rid1 := s.m.GenerateRID()
	extractedNode1 := s.m.GetNodeFromRID(rid1.Int64())
	s.Equal(int64(s.node), extractedNode1, "Extracted node should match initialized node 1")

	// Generate a RID from node 20 (s.m2)
	rid2 := s.m2.GenerateRID()
	extractedNode2 := s.m2.GetNodeFromRID(rid2.Int64())
	s.Equal(int64(s.node2), extractedNode2, "Extracted node should match initialized node 20")
}

func (s *randSuite) TestGenRandomString() {
	testcase := []struct {
		Case      string
		StringLen uint
	}{
		{
			Case:      "Node 1, string len 25",
			StringLen: 25,
		},
	}

	for idx, c := range testcase {
		randStr := s.m.GenRandomString(c.StringLen)
		s.Equal(int(c.StringLen), len(randStr), fmt.Sprintf("Case %d: generate rand string length not match", idx))
	}
}

func (s *randSuite) TestIsValidateRID() {
	n1 := s.m.GenerateRID().Int64()
	n2 := s.m2.GenerateRID().Int64()
	f1 := s.futureM.GenerateRID().Int64()
	tCases := []struct {
		Case         string
		InputRID     int64
		ExpRes       bool
		ExpNode      int64
		ExpRes2      bool
		ExpFutureRes bool
	}{
		{
			Case:         "1",
			InputRID:     n1,
			ExpRes:       true,
			ExpNode:      1,
			ExpRes2:      false,
			ExpFutureRes: false,
		},
		{
			Case:         "2",
			InputRID:     n2,
			ExpRes:       false,
			ExpNode:      20,
			ExpRes2:      true,
			ExpFutureRes: false,
		},
		{
			Case:         "3 future time caused negative RID",
			InputRID:     f1,
			ExpRes:       false,
			ExpNode:      1,
			ExpRes2:      false,
			ExpFutureRes: false, // future time will caused generated a negative RID, but from the IsValidateRID method, it will not validate the RID from the future, there for it will return the false

		},
	}

	for i, c := range tCases {
		fromNode := s.m.GetNodeFromRID(c.InputRID)
		res := s.m.IsValidateRID(c.InputRID)
		s.Equal(c.ExpRes, res, fmt.Sprintf("Case[%d] The node 1 validate RID result is not as expected", i))
		s.Equal(c.ExpNode, fromNode, fmt.Sprintf("Case[%d] The node 1 validate RID result is not as expected", i))
		res2 := s.m2.IsValidateRID(c.InputRID)
		s.Equal(c.ExpRes2, res2, fmt.Sprintf("Case[%d] The node 20 validate RID result is not as expected", i))
		s.Equal(c.ExpFutureRes, s.futureM.IsValidateRID(c.InputRID), fmt.Sprintf("Case[%d] The node 1 from passed time validate RID result is not as expected", i))
	}
}
