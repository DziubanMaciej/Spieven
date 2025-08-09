package packet

import "supervisor/common/types"

type ScheduleRequestBody struct {
	Cmdline               []string
	Cwd                   string
	Env                   []string
	FriendlyName          string
	CaptureStdout         bool
	Display               types.DisplaySelection
	DelayAfterSuccessMs   int
	DelayAfterFailureMs   int
	MaxSubsequentFailures int
}

func EncodeSchedulePacket(data ScheduleRequestBody) (Packet, error) {
	return EncodePacket(PacketIdSchedule, data)
}

func DecodeSchedulePacket(packet Packet) (result ScheduleRequestBody, err error) {
	err = DecodePacket(packet, PacketIdSchedule, &result)
	return
}

type ScheduleResponseBody struct {
	Status  types.ScheduleResponseStatus
	Id      int
	LogFile string
}

func EncodeScheduleResponsePacket(value ScheduleResponseBody) (Packet, error) {
	return EncodePacket(PacketIdScheduleResponse, &value)
}

func DecodeScheduleResponsePacket(packet Packet) (result ScheduleResponseBody, err error) {
	err = DecodePacket(packet, PacketIdScheduleResponse, &result)
	return
}
