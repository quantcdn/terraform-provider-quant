package utils

import (
	"errors"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func GetRuleImportId(s string) (types.String, types.String, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return types.StringNull(), types.StringNull(), errors.New("The ID must follow the pattern project/uuid to import")
	}

	re := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	if !re.MatchString(parts[1]) {
		return types.StringNull(), types.StringNull(), errors.New("Invalid UUID format")
	}

	return types.StringValue(parts[0]), types.StringValue(parts[1]), nil
}
