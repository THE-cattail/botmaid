package botmaid

import (
	"fmt"
	"reflect"
)

// Contains checks if the element is in the slice.
func Contains(s interface{}, a interface{}) bool {
	if reflect.TypeOf(s).Kind() == reflect.Slice {
		t := reflect.ValueOf(s)
		for i := 0; i < t.Len(); i++ {
			if t.Index(i).Interface() == a {
				return true
			}
		}
		return false
	}

	return false
}

// ListToString convert the list to a string.
func ListToString(list []string, format string, separator string, and string) string {
	if len(list) < 1 {
		return ""
	}
	if len(list) == 1 {
		return fmt.Sprintf(format, list[0])
	}
	ret := fmt.Sprintf(format, list[0])
	for i := 1; i < len(list)-1; i++ {
		ret += separator + fmt.Sprintf(format, list[i])
	}
	ret += and + fmt.Sprintf(format, list[len(list)-1])
	return ret
}
