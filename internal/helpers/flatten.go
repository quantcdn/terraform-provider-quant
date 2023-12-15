package helpers

import "github.com/hashicorp/terraform-plugin-framework/types"

func FlattenToInt32(list []types.Int64) []int32 {
	var i []int32
	for _, in := range list {
		i = append(i, int32(in.ValueInt64()))
	}
	return i
}

func FlattenToInt64(list []types.Int64) []int64 {
	var i []int64
	for _, in := range list {
		i = append(i, in.ValueInt64())
	}
	return i
}

func FlattenToStrings(list []types.String) []string {
	var s []string
	for _, str := range list {
		s = append(s, str.ValueString())
	}
	return s
}
