package common

import (
	"os"
	"supervisor/common/types"
)

func SetDisplayEnvVarsForCurrentProcess(display types.DisplaySelection) error {
	setenv := func(name string, value string) error {
		return os.Setenv(name, value)
	}
	unsetenv := func(name string) error {
		return os.Unsetenv(name)
	}
	return SetDisplayEnvVars(display, setenv, unsetenv)
}

func SetDisplayEnvVarsForSubprocess(display types.DisplaySelection, outEnv *[]string) error {
	setenv := func(name string, value string) error {
		*outEnv = append(*outEnv, name+"="+value)
		return nil
	}
	unsetenv := func(name string) error {
		*outEnv = append(*outEnv, name+"=")
		return nil
	}
	return SetDisplayEnvVars(display, setenv, unsetenv)
}

func SetDisplayEnvVars(display types.DisplaySelection, setenv func(string, string) error, unsetenv func(string) error) (err error) {
	if display.Type == types.DisplaySelectionTypeXorg {
		err = setenv("DISPLAY", display.Name)
	} else {
		err = unsetenv("DISPLAY")
	}
	if err != nil {
		return
	}

	if display.Type == types.DisplaySelectionTypeWayland {
		err = setenv("WAYLAND_DISPLAY", display.Name)
	} else {
		err = unsetenv("WAYLAND_DISPLAY")
	}

	return
}
