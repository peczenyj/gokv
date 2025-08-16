package util

import (
	"errors"
)

// CheckKeyAndValue returns an error if k == "" or if v == nil
func CheckKeyAndValue(k string, v any) error {
	if err := CheckKey(k); err != nil {
		return err
	}
	return CheckVal(v)
}

// CheckKey returns an error if k == ""
func CheckKey(k string) error {
	if k == "" {
		return errors.New("the passed key is an empty string, which is invalid")
	}
	return nil
}

// CheckVal returns an error if v == nil
func CheckVal(v any) error {
	if v == nil {
		return errors.New("the passed value is nil, which is not allowed")
	}
	return nil
}
