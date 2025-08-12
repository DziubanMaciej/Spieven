//go:build !user && !release

package buildopts

import (
	"fmt"
	"os"
)

const VersionSuffix = "-dev"
const DefaultPort = "13128"
const AutorunBackend = false
const HandshakeValidationEnabled = false

func PrintBuildFlavourNotice() {
	fmt.Fprintln(os.Stderr, "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	fmt.Fprintln(os.Stderr, "!!! You are using a dev build meant for testing purposes. Dev builds can have !!!")
	fmt.Fprintln(os.Stderr, "!!! some features disabled for ease of debugging. To use a fully functional   !!!")
	fmt.Fprintln(os.Stderr, "!!! Spieven, use an official release or build with -tags user                 !!!")
	fmt.Fprintln(os.Stderr, "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	fmt.Fprintln(os.Stderr, "")
}
