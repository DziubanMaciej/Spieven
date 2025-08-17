package types

import (
	"errors"
	"fmt"
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

type DisplaySelection struct {
	Type DisplaySelectionType
	Name string
}

const DisplaySelectionHelpString = "Use \"h\" for headless, \"x$DISPLAY\" for xorg or \"w$WAYLAND_DISPLAY\" for wayland."

func (display *DisplaySelection) ParseDisplaySelection(val string) error {
	if val == "" {
		display.Type = DisplaySelectionTypeNone
		display.Name = ""
		return nil
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

	display.Name = val[1:]
	if display.Name == "" {
		return errors.New("invalid display selection - it must contain display name")
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
