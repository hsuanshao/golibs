package validator

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type valiatorSuite struct {
	suite.Suite
}

func (s *valiatorSuite) TestIsValidCountryCallingCode() {
	testcase := []struct {
		Desc        string
		CountryCode string
		ExpRes      bool
	}{
		{
			Desc:        "wrong format input",
			CountryCode: "8a",
			ExpRes:      false,
		},
		{
			Desc:        "0 input",
			CountryCode: "0",
			ExpRes:      false,
		},
		{
			Desc:        "1000 input",
			CountryCode: "1000",
			ExpRes:      false,
		},
		{
			Desc:        "886 input",
			CountryCode: "886",
			ExpRes:      true,
		},
		{
			Desc:        "284 input",
			CountryCode: "284",
			ExpRes:      true,
		},
	}

	for _, c := range testcase {
		s.Equal(c.ExpRes, IsValidCountryCallingCode(c.CountryCode), c.Desc)
	}
}

func (s *valiatorSuite) TestIsValidLocalPhoneNumber() {
	testcase := []struct {
		Desc   string
		Phone  string
		ExpRes bool
	}{
		{
			Desc:   "empty input",
			Phone:  "",
			ExpRes: false,
		},
		{
			Desc:   "letter input",
			Phone:  "a",
			ExpRes: false,
		},
		{
			Desc:   "phone number input",
			Phone:  "987654321",
			ExpRes: true,
		},
		{
			Desc:   "phone number input",
			Phone:  "0987654321",
			ExpRes: true,
		},
	}

	for _, c := range testcase {
		s.Equal(c.ExpRes, IsValidLocalPhoneNumber(c.Phone), c.Desc)
	}
}

func (s *valiatorSuite) TestIsValidTwoDigitISO() {
	testcase := []struct {
		Desc      string
		DigitISO  string
		ExpResult bool
	}{
		{
			Desc:      "empty input expect false",
			DigitISO:  "",
			ExpResult: false,
		},
		{
			Desc:      "input a expect false",
			DigitISO:  "a",
			ExpResult: false,
		},
		{
			Desc:      "input abc expect false",
			DigitISO:  "abc",
			ExpResult: false,
		},
		{
			Desc:      "input tw expect true",
			DigitISO:  "tw",
			ExpResult: true,
		},
		{
			Desc:      "input Jp expect true",
			DigitISO:  "Jp",
			ExpResult: true,
		},
		{
			Desc:      "input AB expect true",
			DigitISO:  "AB",
			ExpResult: true,
		},
	}

	for _, c := range testcase {
		s.Equal(c.ExpResult, IsValidTwoDigitISO(c.DigitISO), c.Desc)
	}
}

func (s *valiatorSuite) TestIsValidOpenID() {
	openID := ""
	s.False(IsValidOpenID(openID))

	openID = "a"
	s.False(IsValidOpenID(openID))

	openID = "ab"
	s.True(IsValidOpenID(openID))

	openID = "AB"
	s.True(IsValidOpenID(openID))

	openID = "12345678901234567890"
	s.True(IsValidOpenID(openID))

	openID = "123456789012345678901"
	s.False(IsValidOpenID(openID))

	s.True(IsValidOpenID("棒棒的"))
	s.True(IsValidOpenID("萌萌達"))
	s.True(IsValidOpenID("从头开始"))
	s.True(IsValidOpenID("コーヒー"))
	s.True(IsValidOpenID("圖图図Picture"))
	s.True(IsValidOpenID("从_世_せ_セ_365"))

	s.False(IsValidOpenID("世"))
	s.False(IsValidOpenID("H365世界"))
	s.False(IsValidOpenID("世 界"))
	s.False(IsValidOpenID("世界せいかい"))
	s.False(IsValidOpenID("세상"))
	s.False(IsValidOpenID("😂😂"))
	s.False(IsValidOpenID("ＸＤ"))
	s.False(IsValidOpenID("１２３４"))
	s.False(IsValidOpenID("ｴｴｴｴ"))
	s.False(IsValidOpenID("Pokémon"))
}

func (s *valiatorSuite) TestIsInStringSlice() {
	testcase := []struct {
		Desc   string
		Slice  []string
		Word   string
		ExpRes bool
	}{
		{
			Desc:   "normal test",
			Slice:  []string{"abc", "edc", "happy", "GFW"},
			Word:   "GFW",
			ExpRes: true,
		},
		{
			Desc:   "not exists test",
			Slice:  []string{"abc", "edc", "happy", "GFW"},
			Word:   "US",
			ExpRes: false,
		},
	}

	for _, c := range testcase {
		s.Equal(c.ExpRes, IsInStringSlice(c.Slice, c.Word), c.Desc)
	}
}

func (s *valiatorSuite) TestIsInIntSlice() {
	testcase := []struct {
		Desc   string
		Slice  []int
		Word   int
		ExpRes bool
	}{
		{
			Desc:   "normal test",
			Slice:  []int{1, 7, 3, 8},
			Word:   8,
			ExpRes: true,
		},
		{
			Desc:   "not exists test",
			Slice:  []int{1, 7, 3, 8},
			Word:   5111,
			ExpRes: false,
		},
	}

	for _, c := range testcase {
		s.Equal(c.ExpRes, IsInIntSlice(c.Slice, c.Word), c.Desc)
	}
}

func (s *valiatorSuite) TestIsInInt64Slice() {
	testcase := []struct {
		Desc   string
		Slice  []int64
		Word   int64
		ExpRes bool
	}{
		{
			Desc:   "normal test",
			Slice:  []int64{23412343342231, 234231, 32323223134, 812342132434},
			Word:   234231,
			ExpRes: true,
		},
		{
			Desc:   "not exists test",
			Slice:  []int64{1, 7, 3, 8},
			Word:   5111,
			ExpRes: false,
		},
	}

	for _, c := range testcase {
		s.Equal(c.ExpRes, IsInInt64Slice(c.Slice, c.Word), c.Desc)
	}
}

func TestSmsSuite(t *testing.T) {
	suite.Run(t, new(valiatorSuite))
}
