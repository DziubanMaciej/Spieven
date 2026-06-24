package packet

import "spieven/common/types"

type RunRequestBody struct {
	Cmdline               []string
	Cwd                   string
	Env                   []string
	FriendlyName          string
	CaptureStdout         bool
	CaptureStderr         bool
	Display               types.DisplaySelection
	DelayAfterSuccessMs   int
	DelayAfterFailureMs   int
	MaxSubsequentFailures int
	Tags                  []string
}

func EncodeRunPacket(data RunRequestBody) (Packet, error) {
	return EncodePacket(PacketIdRun, data)
}

func DecodeRunPacket(packet Packet) (result RunRequestBody, err error) {
	err = DecodePacket(packet, PacketIdRun, &result)
	return
}

type RunResponseBody struct {
	Status  types.RunResponseStatus
	Id      int
	LogFile string
}

func EncodeRunResponsePacket(value RunResponseBody) (Packet, error) {
	return EncodePacket(PacketIdRunResponse, &value)
}

func DecodeRunResponsePacket(packet Packet) (result RunResponseBody, err error) {
	err = DecodePacket(packet, PacketIdRunResponse, &result)
	return
}
