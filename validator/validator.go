package validator

import "regexp"

// IsValidCountryCallingCode returns if countryCallingCode format is valid
// Example: 886, 81
func IsValidCountryCallingCode(countryCallingCode string) bool {
	return regexp.MustCompile(`^[1-9][0-9]{0,2}$`).MatchString(countryCallingCode)
}

// IsValidLocalPhoneNumber returns if localPhoneNumber format is valid
// Example: 0987654321, 987654321
func IsValidLocalPhoneNumber(localPhoneNumber string) bool {
	return regexp.MustCompile(`^[0-9]+$`).MatchString(localPhoneNumber)
}

// IsValidTwoDigitISO returns if twoDigitISO format is valid
// Example: tw, JP
func IsValidTwoDigitISO(twoDigitISO string) bool {
	return regexp.MustCompile(`^[a-zA-Z]{2}$`).MatchString(twoDigitISO)
}

// IsValidOpenID returns if user openID format is valid
func IsValidOpenID(openID string) bool {
	return regexp.MustCompile(`^[\p{Han}\x{3041}-\x{3096}\x{30A1}-\x{30FC}\w.]{2,20}$`).MatchString(openID) && len(regexp.MustCompile(`[\p{Han}\x{3041}-\x{3096}\x{30A1}-\x{30FC}]`).FindAllStringIndex(openID, -1)) <= 4 && !regexp.MustCompile(`^H365`).MatchString(openID)
}

// IsInStringSlice validates target exists in list or not
func IsInStringSlice(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

// IsInIntSlice validates target exists in list or not
func IsInIntSlice(intlist []int, target int) bool {
	for _, s := range intlist {
		if s == target {
			return true
		}
	}
	return false
}

// IsInInt64Slice validates target exists in list or not
func IsInInt64Slice(int64list []int64, target int64) bool {
	for _, s := range int64list {
		if s == target {
			return true
		}
	}
	return false
}
