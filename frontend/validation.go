package frontend

import (
	"errors"
	"fmt"
	"spieven/common/packet"
	"unicode"
)

type ValidationType byte

const (
	ValidationTypeGeneric ValidationType = iota
	ValidationTypeAlphanumeric
)

func ValidateString(val string, stringName string, validationType ValidationType) error {
	for _, r := range val {
		switch validationType {
		case ValidationTypeGeneric:
			if unicode.IsControl(r) {
				return fmt.Errorf("%v contains invalid characters. Whitespace and control characters are forbidden", stringName)
			}
		case ValidationTypeAlphanumeric:
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-') {
				return fmt.Errorf("%v contains invalid characters. Only alphanumeric characters, hyphens and underscore are allowed", stringName)
			}
		default:
			return errors.New("invalid validation type")
		}

	}
	return nil
}

func ValidateStrings(val []string, stringName string, validationType ValidationType) error {
	for _, str := range val {
		err := ValidateString(str, stringName, validationType)
		if err != nil {
			return err
		}
	}
	return nil
}

func ValidateScheduleRequestBody(val *packet.ScheduleRequestBody) error {
	if err := ValidateString(val.Cwd, "field cwd", ValidationTypeGeneric); err != nil {
		return err
	}
	if err := ValidateString(val.FriendlyName, "field friendlyName", ValidationTypeAlphanumeric); err != nil {
		return err
	}
	if err := ValidateStrings(val.Cmdline, "field cmdline", ValidationTypeGeneric); err != nil {
		return err
	}
	if err := ValidateStrings(val.Env, "field env", ValidationTypeGeneric); err != nil {
		return err
	}
	if err := ValidateStrings(val.Tags, "field tags", ValidationTypeAlphanumeric); err != nil {
		return err
	}
	return nil
}
