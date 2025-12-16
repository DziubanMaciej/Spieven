package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type DisplaySelectionType byte

const (
	DisplaySelectionTypeNone DisplaySelectionType = iota
	DisplaySelectionTypeHeadless
	DisplaySelectionTypeXorg
	DisplaySelectionTypeWayland
)

func (t DisplaySelectionType) String() string {
	switch t {
	case DisplaySelectionTypeNone:
		return "none"
	case DisplaySelectionTypeHeadless:
		return "headless"
	case DisplaySelectionTypeXorg:
		return "xorg"
	case DisplaySelectionTypeWayland:
		return "wayland"
	default:
		return "invalid"
	}
}

func (t DisplaySelectionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *DisplaySelectionType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case "none":
		*t = DisplaySelectionTypeNone
	case "headless":
		*t = DisplaySelectionTypeHeadless
	case "xorg":
		*t = DisplaySelectionTypeXorg
	case "wayland":
		*t = DisplaySelectionTypeWayland
	default:
		return fmt.Errorf("invalid DisplaySelectionType: %q", s)
	}

	return nil
}

type DisplaySelection struct {
	Type DisplaySelectionType
	Name string
}

const DisplaySelectionHelpString = "Use \"h\" for headless, \"x\" for xorg or \"w\" for wayland. You can also select a specific display with \"x:0\" or \"wwayland-1\"."

func (display *DisplaySelection) ParseDisplaySelection(val string, allowNone bool) error {
	if len(val) == 0 {
		if allowNone {
			display.Type = DisplaySelectionTypeNone
			display.Name = ""
			return nil
		} else {
			return errors.New("invalid display selection - it must not be empty")
		}
	}

	switch val[0] {
	case 'h':
		display.Type = DisplaySelectionTypeHeadless
		display.Name = ""
		return nil
	case 'x':
		display.Type = DisplaySelectionTypeXorg
	case 'w':
		display.Type = DisplaySelectionTypeWayland
	default:
		return errors.New("invalid display selection - it must be either headless, xorg or wayland")
	}

	if len(val) == 1 {
		// Derive display name from env
		name, err := readCurrentDisplayNameFromEnv(display.Type)
		if err != nil {
			return fmt.Errorf("%v; please specify display name explicitly", err.Error())
		}
		display.Name = name
	} else {
		// Explicity passed display name
		if display.Type == DisplaySelectionTypeHeadless {
			return errors.New("invalid display selection - headless display cannot have a name")
		}
		display.Name = val[1:]
	}

	return nil
}

func (display *DisplaySelection) ComputeDisplayLabel() string {
	switch display.Type {
	case DisplaySelectionTypeHeadless:
		return "h"
	case DisplaySelectionTypeXorg:
		return fmt.Sprintf("x%v", display.Name)
	case DisplaySelectionTypeWayland:
		return fmt.Sprintf("w%v", display.Name)
	default:
		return "unknown"
	}
}

func (display *DisplaySelection) ComputeDisplayLabelLong() string {
	switch display.Type {
	case DisplaySelectionTypeHeadless:
		return "headless"
	case DisplaySelectionTypeXorg:
		return fmt.Sprintf("xorg %v", display.Name)
	case DisplaySelectionTypeWayland:
		return fmt.Sprintf("wayland %v", display.Name)
	default:
		return "unknown"
	}
}

func readCurrentDisplayNameFromEnv(displayType DisplaySelectionType) (string, error) {
	var envName string

	switch displayType {
	case DisplaySelectionTypeXorg:
		envName = "DISPLAY"
	case DisplaySelectionTypeWayland:
		envName = "WAYLAND_DISPLAY"
	case DisplaySelectionTypeHeadless:
		return "", nil
	default:
		return "", errors.New("invalid display type")
	}

	displayName, found := os.LookupEnv(envName)
	if !found {
		return "", fmt.Errorf("failed to read %v env", envName)
	}
	if displayName == "" {
		return "", fmt.Errorf("%v is empty", envName)
	}
	return displayName, nil
}
