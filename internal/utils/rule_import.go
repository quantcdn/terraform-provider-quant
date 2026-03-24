package utils

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func GetRuleImportId(s string) (types.String, types.String, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return types.StringNull(), types.StringNull(), errors.New("the ID must follow the pattern project/uuid to import")
	}

	re := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	if !re.MatchString(parts[1]) {
		return types.StringNull(), types.StringNull(), errors.New("invalid UUID format")
	}

	return types.StringValue(parts[0]), types.StringValue(parts[1]), nil
}

func GetDomainImportId(s string) (types.String, types.Int64, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return types.StringNull(), types.Int64Null(), errors.New("the ID must follow the pattern project/uuid to import")
	}

	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return types.StringNull(), types.Int64Null(), errors.New("invalid domain ID format")
	}

	return types.StringValue(parts[0]), types.Int64Value(id), nil
}
