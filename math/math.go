package math

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/hsuanshao/golibs/ctx"
	"github.com/sirupsen/logrus"
)

func New() Math {
	return &impl{}
}

type impl struct{}

// MaxInt64 returns the bigger int64
func (m *impl) MaxInt64(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}

// MinInt64 returns the smaller int64
func (m *impl) MinInt64(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

// MaxInt returns the bigger int
func (m *impl) MaxInt(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// MinInt returns the smaller int
func (m *impl) MinInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// MaxFloat64 returns the bigger float64
func (m *impl) MaxFloat64(x, y float64) float64 {
	if x < y {
		return y
	}
	return x
}

// MinFloat64 returns the smaller float64
func (m *impl) MinFloat64(x, y float64) float64 {
	if x < y {
		return x
	}
	return y
}

// ConvertInt64ToHexHexadecimal to transfer Decimal number to Hexadecimal number (but data type as string, and it has 0x prefix)
func (m *impl) ConvertInt64ToHexadecimal(ctx ctx.CTX, num int64) (hexadecimalNum string) {
	str16 := strconv.FormatInt(num, 16)
	return "0x" + str16
}

// ConvertDecimalStrToHexadecimalStr to transfer decimal number string to hexadecimal string
func (m *impl) ConvertDecimalStrToHexadecimalStr(ctx ctx.CTX, decimalNum string) (hexadecimalNum string, err error) {
	var ok bool
	bigNum := new(big.Int)
	bigNum, ok = bigNum.SetString(decimalNum, 10)
	if !ok {
		ctx.WithFields(logrus.Fields{"is validated decimal": ok, "input decimal": decimalNum}).Error("convert input decimal number failed")
		return "", ErrIncorrectDecimalFormat
	}

	return fmt.Sprintf("%x", bigNum), nil
}

// ConvertHexadecimalStrToDecimalStr to transfer hexadecimal number string to decimal string
func (m *impl) ConvertHexadecimalStrToDecimalStr(ctx ctx.CTX, hexadecimalNum string) (decimalNum string, err error) {
	foundCase1 := strings.HasPrefix(hexadecimalNum, "0x")
	foundCase2 := strings.HasPrefix(hexadecimalNum, "0X")
	if foundCase1 || foundCase2 {
		hexadecimalNum = hexadecimalNum[2:]
	}
	var ok bool
	bigNum := new(big.Int)
	bigNum, ok = bigNum.SetString(hexadecimalNum, 16)
	if !ok {
		hasPrefixOx := false
		if foundCase1 || foundCase2 {
			hasPrefixOx = true
		}
		ctx.WithFields(logrus.Fields{"hexadecimalNumStr": hexadecimalNum, "hasPrefixOx": hasPrefixOx, "is validate hexadecimal number": ok}).Error("convert hexadecimal to decimal format string failed")
		return "", ErrIncorrectHexadecimalFormat
	}

	return bigNum.String(), nil
}

// ConvertBigIntToHexadecimal to transfer big.Int to Hexadecimal number string
func (m *impl) ConvertBigIntToHexadecimal(ctx ctx.CTX, bigNum *big.Int) (hexadecimalNum string) {
	hexNum := fmt.Sprintf("%x", bigNum)
	ctx.WithFields(logrus.Fields{"num": bigNum.String(), "hexadecimal": hexNum}).Info("convert check")
	return hexNum
}

// ConvertHexadecimalToInt64 to convert hexadecimal number to decimal number (int64)
func (m *impl) ConvertHexadecimalToInt64(ctx ctx.CTX, hexadecimalNum string) (decimalNum *int64, err error) {
	foundCase1 := strings.HasPrefix(hexadecimalNum, "0x")
	foundCase2 := strings.HasPrefix(hexadecimalNum, "0X")
	if foundCase1 || foundCase2 {
		hexadecimalNum = hexadecimalNum[2:]
	}

	decimalRes, err := strconv.ParseInt(hexadecimalNum, 16, 64)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "hexadecimalNum": hexadecimalNum}).Error("convert hexadecimal to int64 decimal number get error")
		return nil, ErrParseHexadecimalFailed
	}

	return &decimalRes, nil
}

// ConvertHexadecimalToBigNum purpose same as ConvertHexadecimalToInt64, but for the case if the number to large, therefor, it handle by big.Int
func (m *impl) ConvertHexadecimalToBigNum(ctx ctx.CTX, hexadecimalNum string) (num *big.Int, err error) {
	var ok bool
	foundCase1 := strings.HasPrefix(hexadecimalNum, "0x")
	foundCase2 := strings.HasPrefix(hexadecimalNum, "0X")
	if foundCase1 || foundCase2 {
		hexadecimalNum = hexadecimalNum[2:]
	}

	bigNum := new(big.Int)
	bigNum, ok = bigNum.SetString(hexadecimalNum, 16)
	if !ok {
		ctx.WithFields(logrus.Fields{"input string": hexadecimalNum}).Warn("unable convert hex string to big int")
		return nil, ErrConvertHexadecimalToBigInt
	}

	return bigNum, nil
}
