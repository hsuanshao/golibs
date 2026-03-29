package math

import (
	"math/big"

	"github.com/hsuanshao/golibs/ctx"
)

type Math interface {
	// MaxInt64 returns the bigger int64
	MaxInt64(x, y int64) int64
	// MinInt64 returns the smaller int64
	MinInt64(x, y int64) int64
	// MaxInt returns the bigger int
	MaxInt(x, y int) int
	// MinInt returns the smaller int
	MinInt(x, y int) int
	// MinFloat64 returns the smaller float64
	MaxFloat64(x, y float64) float64
	// MinFloat64 returns the smaller float64
	MinFloat64(x, y float64) float64

	// ConvertInt64ToHexadecimal to transfer Decimal number to Hexadecimal number (but data type as string, and it has 0x prefix)
	ConvertInt64ToHexadecimal(ctx ctx.CTX, num int64) (hexadecimalNum string)

	// ConvertDecimalStrToHexadecimalStr to transfer decimal number string to hexadecimal string
	ConvertDecimalStrToHexadecimalStr(ctx ctx.CTX, decimalNum string) (hexadecimalNum string, err error)
	// ConvertHexadecimalStrToDecimalStr to transfer hexadecimal number string to decimal string
	ConvertHexadecimalStrToDecimalStr(ctx ctx.CTX, hexadecimalNum string) (decimalNum string, err error)
	// ConvertBigIntToHexadecimal to transfer big.Int to Hexadecimal number string
	ConvertBigIntToHexadecimal(ctx ctx.CTX, bigNum *big.Int) (hexadecimalNum string)
	// ConvertHexadecimalToInt64 to convert hexadecimal number to decimal number (int64)
	ConvertHexadecimalToInt64(ctx ctx.CTX, hexadecimalNum string) (decimalNum *int64, err error)
	// ConvertHexadecimalToBigNum purpose same as ConvertHexadecimalToInt64, but for the case if the number to large, therefor, it handle by big.Int
	ConvertHexadecimalToBigNum(ctx ctx.CTX, hexadecimalNum string) (num *big.Int, err error)
}
