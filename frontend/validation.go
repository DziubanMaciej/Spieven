package frontend

import (
	"fmt"
	"spieven/common/packet"
	"unicode"
)

func ValidateString(val string, stringName string) error {
	for _, r := range val {
		if unicode.IsControl(r) || r == '"' || r == '\'' || r == '\\' {
			return fmt.Errorf("%v contains invalid characters", stringName)
		}
	}
	return nil
}

func ValidateStrings(val []string, stringName string) error {
	for _, str := range val {
		err := ValidateString(str, stringName)
		if err != nil {
			return err
		}
	}
	return nil
}

func ValidateScheduleRequestBody(val *packet.ScheduleRequestBody) error {
	if err := ValidateString(val.Cwd, "field cwd"); err != nil {
		return err
	}
	if err := ValidateString(val.FriendlyName, "field friendlyName"); err != nil {
		return err
	}
	if err := ValidateStrings(val.Cmdline, "field cmdline"); err != nil {
		return err
	}
	if err := ValidateStrings(val.Env, "field env"); err != nil {
		return err
	}
	return nil
}
