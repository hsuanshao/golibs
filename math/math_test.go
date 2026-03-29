package math

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/hsuanshao/golibs/ctx"
)

var (
	mockCTX = ctx.Background()
)

func TestMath(t *testing.T) {
	ctx.SetDebugLevel()
	suite.Run(t, new(mathSuite))
}

type mathSuite struct {
	suite.Suite
	Math Math
}

func (ms *mathSuite) SetupTest() {
	ms.Math = &impl{}
}

func (ms *mathSuite) TearDownTest() {}

func (ms *mathSuite) TestMaxInt64() {
	testcases := []struct {
		Case   string
		InputX int64
		InputY int64
		ExpRes int64
	}{
		{
			Case:   "-100 vs 100",
			InputX: -100,
			InputY: 100,
			ExpRes: 100,
		},
		{
			Case:   "100009 vs 100010",
			InputX: 100010,
			InputY: 100009,
			ExpRes: 100010,
		},
	}

	for cIdx, c := range testcases {
		res := ms.Math.MaxInt64(c.InputX, c.InputY)
		ms.Equal(c.ExpRes, res, fmt.Sprintf("MaxInt64: case %d: %s", cIdx, c.Case))
	}
}

func (ms *mathSuite) TestMinInt64() {
	testcases := []struct {
		Case   string
		InputX int64
		InputY int64
		ExpRes int64
	}{
		{
			Case:   "-100 vs 100",
			InputX: -12893407623400,
			InputY: 100,
			ExpRes: -12893407623400,
		},
		{
			Case:   "100009 vs 100010",
			InputX: 100010,
			InputY: 100009,
			ExpRes: 100009,
		},
	}

	for cIdx, c := range testcases {
		res := ms.Math.MinInt64(c.InputX, c.InputY)
		ms.Equal(c.ExpRes, res, fmt.Sprintf("MinInt64: case %d: %s", cIdx, c.Case))
	}
}

func (ms *mathSuite) TestMaxInt() {
	testcases := []struct {
		Case   string
		InputX int
		InputY int
		ExpRes int
	}{
		{
			Case:   "-100 vs 100",
			InputX: -100,
			InputY: 100,
			ExpRes: 100,
		},
		{
			Case:   "100009 vs 100010",
			InputX: 100010,
			InputY: 100009,
			ExpRes: 100010,
		},
	}

	for cIdx, c := range testcases {
		res := ms.Math.MaxInt(c.InputX, c.InputY)
		ms.Equal(c.ExpRes, res, fmt.Sprintf("MaxInt: case %d: %s", cIdx, c.Case))
	}
}

func (ms *mathSuite) TestMinInt() {
	testcases := []struct {
		Case   string
		InputX int
		InputY int
		ExpRes int
	}{
		{
			Case:   "-100 vs 100",
			InputX: -100,
			InputY: 100,
			ExpRes: -100,
		},
		{
			Case:   "100009 vs 100010",
			InputX: 100010,
			InputY: 100009,
			ExpRes: 100009,
		},
	}

	for cIdx, c := range testcases {
		res := ms.Math.MinInt(c.InputX, c.InputY)
		ms.Equal(c.ExpRes, res, fmt.Sprintf("MinInt: case %d: %s", cIdx, c.Case))
	}
}

func (ms *mathSuite) TestMaxFloat64() {
	testcases := []struct {
		Case   string
		InputX float64
		InputY float64
		ExpRes float64
	}{
		{
			Case:   "100.1425926154 vs 100.0000123485",
			InputX: 100.1425926154,
			InputY: 100.0000123485,
			ExpRes: 100.1425926154,
		},
	}

	for cIdx, c := range testcases {
		res := ms.Math.MaxFloat64(c.InputX, c.InputY)
		ms.Equal(c.ExpRes, res, fmt.Sprintf("MaxFloat64: case %d: %s", cIdx, c.Case))
	}
}

func (ms *mathSuite) TestMinFloat64() {
	testcases := []struct {
		Case   string
		InputX float64
		InputY float64
		ExpRes float64
	}{
		{
			Case:   "100.1425926154 vs 100.0000123485",
			InputX: 100.1425926154,
			InputY: 100.0000123485,
			ExpRes: 100.0000123485,
		},
	}

	for cIdx, c := range testcases {
		res := ms.Math.MinFloat64(c.InputX, c.InputY)
		ms.Equal(c.ExpRes, res, fmt.Sprintf("MinFloat64: case %d: %s", cIdx, c.Case))
	}
}

func (ms *mathSuite) TestConvertInt64ToHexadecimal() {
	testcases := []struct {
		Case   string
		InputX int64
		ExpRes string
	}{
		{
			Case:   "17113403",
			InputX: 17113403,
			ExpRes: "0x105213b",
		},
		{
			Case:   "255432487123732",
			InputX: 255432487123732,
			ExpRes: "0xe85082a8bb14",
		},
	}

	for cIdx, c := range testcases {
		res := ms.Math.ConvertInt64ToHexadecimal(mockCTX, c.InputX)
		ms.Equal(c.ExpRes, res, fmt.Sprintf("ConvertInt64ToHexadecimal: case %d: %s", cIdx, c.Case))
	}
}

func (ms *mathSuite) TestConvertDecimalStrToHexadecimalStr() {
	testcases := []struct {
		Case   string
		InputX string
		ExpRes string
		ExpErr error
	}{
		{
			Case:   "convert 255432487123732 to hexadecimal",
			InputX: "255432487123732",
			ExpRes: "e85082a8bb14",
			ExpErr: nil,
		},
		{
			Case:   "17113403",
			InputX: "17113403",
			ExpRes: "105213b",
			ExpErr: nil,
		},
		{
			Case:   "Incorrect decimal string",
			InputX: "1A33BD",
			ExpRes: "",
			ExpErr: ErrIncorrectDecimalFormat,
		},
	}

	for cIdx, c := range testcases {
		res, err := ms.Math.ConvertDecimalStrToHexadecimalStr(mockCTX, c.InputX)
		ms.Equal(c.ExpErr, err, fmt.Sprintf("ConvertDecimalStrToHexadecimalStr: case %d: %s", cIdx, c.Case))
		ms.Equal(c.ExpRes, res, fmt.Sprintf("ConvertDecimalStrToHexadecimalStr: case %d: %s", cIdx, c.Case))
	}
}

func (ms *mathSuite) TestConvertHexadecimalStrToDecimalStr() {
	testcases := []struct {
		Case   string
		InputX string
		ExpRes string
		ExpErr error
	}{
		{
			Case:   "convert 0xe85082a8bb14",
			InputX: "0xe85082a8bb14",
			ExpRes: "255432487123732",
			ExpErr: nil,
		},
		{
			Case:   "convert 105213b",
			InputX: "105213b",
			ExpRes: "17113403",
			ExpErr: nil,
		},
		{
			Case:   "Incorrect hexadecimal string",
			InputX: "AXZ8362f935820e554468580210d5b4c000905a1a6060013272982a1422d0a32421c81508a00e1640910411c63027f64c2bd140542c82420601006d004e0891011324f7590618096b0d0009901540e741b9048c8e490122628168a0480634ec019906b0474520124cf0e1bb0380586293b83de94981189a24148a60020a68e80c94a9664",
			ExpRes: "",
			ExpErr: ErrIncorrectHexadecimalFormat,
		},
	}

	for cIdx, c := range testcases {
		res, err := ms.Math.ConvertHexadecimalStrToDecimalStr(mockCTX, c.InputX)
		ms.Equal(c.ExpErr, err, fmt.Sprintf("ConvertHexadecimalStrToDecimalStr: case %d: %s", cIdx, c.Case))
		ms.Equal(c.ExpRes, res, fmt.Sprintf("ConvertHexadecimalStrToDecimalStr: case %d: %s", cIdx, c.Case))
	}
}

func (ms *mathSuite) TestConvertBigIntToHexadecimal() {
	testcases := []struct {
		Case   string
		InputX *big.Int
		ExpRes string
		ExpErr error
	}{
		{
			Case:   "convert 0xe85082a8bb14",
			InputX: big.NewInt(int64(255432487123732)),
			ExpRes: "e85082a8bb14",
			ExpErr: nil,
		},
		{
			Case:   "convert 105213b",
			InputX: big.NewInt(17113403),
			ExpRes: "105213b",
			ExpErr: nil,
		},
	}

	for cIdx, c := range testcases {
		res := ms.Math.ConvertBigIntToHexadecimal(mockCTX, c.InputX)

		ms.Equal(c.ExpRes, res, fmt.Sprintf("ConvertBigIntToHexadecimal: case %d: %s", cIdx, c.Case))
	}
}

func (ms *mathSuite) TestConvertHexadecimalToInt64() {
	testcases := []struct {
		Case   string
		InputX string
		ExpRes int64
		ExpErr error
	}{
		{
			Case:   "convert 0xe85082a8bb14",
			InputX: "0xe85082a8bb14",
			ExpRes: 255432487123732,
			ExpErr: nil,
		},
		{
			Case:   "convert 105213b",
			InputX: "105213b",
			ExpRes: 17113403,
			ExpErr: nil,
		},
		{
			Case:   "Incorrect hexadecimal string",
			InputX: "AXZ8362f935820e554468580210d5b4c000905a1a6060013272982a1422d0a32421c81508a00e1640910411c63027f64c2bd140542c82420601006d004e0891011324f7590618096b0d0009901540e741b9048c8e490122628168a0480634ec019906b0474520124cf0e1bb0380586293b83de94981189a24148a60020a68e80c94a9664",
			ExpErr: ErrParseHexadecimalFailed,
		},
	}

	for cIdx, c := range testcases {
		res, err := ms.Math.ConvertHexadecimalToInt64(mockCTX, c.InputX)
		ms.Equal(c.ExpErr, err, fmt.Sprintf("ConvertHexadecimalToInt64: case %d: %s", cIdx, c.Case))
		if c.ExpErr == nil {
			ms.Equal(c.ExpRes, *res, fmt.Sprintf("ConvertHexadecimalToInt64: case %d: %s", cIdx, c.Case))
		}
	}
}

func (ms *mathSuite) TestConvertHexadecimalToBigNum() {
	testcases := []struct {
		Case   string
		InputX string
		ExpRes *big.Int
		ExpErr error
	}{
		{
			Case:   "convert 0xe85082a8bb14",
			InputX: "0xe85082a8bb14",
			ExpRes: big.NewInt(255432487123732),
			ExpErr: nil,
		},
		{
			Case:   "convert 105213b",
			InputX: "105213b",
			ExpRes: big.NewInt(17113403),
			ExpErr: nil,
		},
		{
			Case:   "Incorrect hexadecimal string",
			InputX: "AXZ8362f935820e554468580210d5b4c000905a1a6060013272982a1422d0a32421c81508a00e1640910411c63027f64c2bd140542c82420601006d004e0891011324f7590618096b0d0009901540e741b9048c8e490122628168a0480634ec019906b0474520124cf0e1bb0380586293b83de94981189a24148a60020a68e80c94a9664",
			ExpErr: ErrConvertHexadecimalToBigInt,
		},
	}

	for cIdx, c := range testcases {
		res, err := ms.Math.ConvertHexadecimalToBigNum(mockCTX, c.InputX)
		ms.Equal(c.ExpErr, err, fmt.Sprintf("ConvertHexadecimalToBigNum: case %d: %s", cIdx, c.Case))
		if c.ExpErr == nil {
			ms.Equal(c.ExpRes.String(), res.String(), fmt.Sprintf("ConvertHexadecimalToBigNum: case %d: %s", cIdx, c.Case))
		}
	}
}
