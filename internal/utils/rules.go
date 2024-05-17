package utils

import (
	"fmt"
)

func GetRuleAny() *string {
	var s string
	s = "any"
	return &s
}

func GetFilterIs(filter string) (*string) {
	var s string
	s = fmt.Sprintf("%s_is", filter)
	return &s
}

func GetFilterIsNot(filter string) (*string) {
	var s string
	s = fmt.Sprintf("%s_is_not", filter)
	return &s
}

func IsValidRuleFilter(s string) bool {
	switch s {
		case "country_is_not", "method_is_not", "ip_is_not":
			return true
		case "country_is", "method_is", "ip_is":
			return true
		case "any":
			return true
	}
	return false
}
