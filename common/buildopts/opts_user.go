//go:build user

package buildopts

const VersionSuffix = "-user"
const DefaultPort = "13130"
const AutorunBackend = true
const HandshakeValidationEnabled = true

func PrintBuildFlavourNotice() {}
