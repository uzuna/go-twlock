package twlock

import "reflect"

// WriteToInterface is utility
// Its cast to responce type and set the responce
func WriteToInterface(res interface{}, v interface{}) error {
	x := reflect.ValueOf(v)
	if x.Kind() == reflect.Ptr {
		x = x.Elem()
	}
	reflect.ValueOf(res).Elem().Set(x)
	return nil
}
