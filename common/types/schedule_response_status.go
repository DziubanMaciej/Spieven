package types

type RunResponseStatus byte

const (
	RunResponseStatusSuccess RunResponseStatus = iota
	RunResponseStatusAlreadyRunning
	RunResponseStatusNameDisplayAlreadyRunning
	RunResponseStatusInvalidDisplay
	RunResponseStatusTaskNotFound       // only for reEncodeRunResponsePacket
	RunResponseStatusTaskNotDeactivated // only for reEncodeRunResponsePacket
	RunResponseStatusUnknown
)
