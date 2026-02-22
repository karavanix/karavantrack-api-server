package utils

import (
	"reflect"
	"strings"
)

func GetTemplateName[T any]() string {
	var t T
	typeName := reflect.TypeOf(t).String()

	// Clean up the type name
	// Remove package path if exists
	if idx := strings.LastIndex(typeName, "."); idx != -1 {
		typeName = typeName[idx+1:]
	}

	// Remove any special characters (like *)
	typeName = strings.TrimPrefix(typeName, "*")

	return typeName
}
