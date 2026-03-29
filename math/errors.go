package math

import "errors"

// define error of math

var (
	ErrParseHexadecimalFailed = errors.New("transfer hexadecimal number string to int64 decimal number failed")

	ErrConvertHexadecimalToBigInt = errors.New("transfer hexadecimal number string to big int decimal number failed")

	// ErrIncorrectDecimalFormat ....
	ErrIncorrectDecimalFormat = errors.New("input decimal number string format is not correct")

	// ErrIncorrectHexadecimalFormat ...
	ErrIncorrectHexadecimalFormat = errors.New("input hexadecimal number string format is not correct")
)
