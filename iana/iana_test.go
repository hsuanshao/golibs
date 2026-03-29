package iana

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/hsuanshao/golibs/ctx"

	ifc "github.com/hsuanshao/golibs/iana/entity/interface"
)

var (
	mockCTX = ctx.Background()
)

type ianaSuite struct {
	suite.Suite
	repository ifc.Repository
}

func TestSuite(t *testing.T) {
	ctx.SetDebugLevel()
	suite.Run(t, new(ianaSuite))
}

func (is *ianaSuite) SetupSuite() {
	if is.repository == nil {
		is.repository = NewRepository(mockCTX)
	}
}

func (is *ianaSuite) TearDownSuite() {
}

func (is *ianaSuite) TestGetTimezoneList() {
	list := is.repository.GetTimezoneList(mockCTX)
	is.Equal(ianaTimeZones, list, "check get timezone list is not equal")
}

func (is *ianaSuite) TestQueryLocation() {
	cases := []struct {
		Case      string
		CTX       ctx.CTX
		QueryName string
		ExpResLen int
		ExpErr    error
	}{
		{
			Case:      "US",
			CTX:       mockCTX,
			QueryName: "US",
			ExpResLen: 29,
			ExpErr:    nil,
		},
		{
			Case:      "Russia",
			CTX:       mockCTX,
			QueryName: "Russia",
			ExpResLen: 26,
			ExpErr:    nil,
		},
		{
			Case:      "Macao",
			CTX:       mockCTX,
			QueryName: "Macao",
			ExpResLen: 1,
			ExpErr:    nil,
		},
		{
			Case:      "Macao code",
			CTX:       mockCTX,
			QueryName: "MO",
			ExpResLen: 1,
			ExpErr:    nil,
		},
		{
			Case:      "query mo",
			CTX:       mockCTX,
			QueryName: "mo",
			ExpResLen: 17,
			ExpErr:    nil,
		},
		{
			Case:      "AR",
			CTX:       mockCTX,
			QueryName: "AR",
			ExpResLen: 12,
			ExpErr:    nil,
		},
		{
			Case:      "Earth",
			CTX:       mockCTX,
			QueryName: "earth",
			ExpResLen: 0,
			ExpErr:    ErrNameNotFound,
		},
	}

	for _, c := range cases {
		c.CTX.WithFields(logrus.Fields{"case": c.Case}).Info("start testing case")
		res, err := is.repository.QueryLocation(c.CTX, c.QueryName)
		is.Equal(c.ExpErr, err)
		if c.ExpErr == nil {
			if !is.Equal(c.ExpResLen, len(res), fmt.Sprintf("case: %s, length not equal", c.Case)) {
				c.CTX.WithField("name", c.QueryName).Warn("check not equal result")
				for _, z := range res {
					zb, _ := json.Marshal(z)
					fmt.Println(string(zb))
				}
			}

		}
	}
}
