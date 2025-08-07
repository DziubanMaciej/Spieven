package packet

type ScheduleRequestBody struct {
	Cmdline       []string
	Cwd           string
	Env           []string
	UserIndex     int
	FriendlyName  string
	CaptureStdout bool
	// TODO add MaxSubsequentFailures int (-1 for no limit)
	// TODO add interval between runs
}

func EncodeSchedulePacket(data ScheduleRequestBody) (Packet, error) {
	return EncodePacket(PacketIdSchedule, data)
}

func DecodeSchedulePacket(packet Packet) (result ScheduleRequestBody, err error) {
	err = DecodePacket(packet, PacketIdSchedule, &result)
	return
}

type ScheduleResponseStatus byte

const (
	ScheduleResponseStatusSuccess ScheduleResponseStatus = iota
	ScheduleResponseStatusAlreadyRunning
	ScheduleResponseStatusNameDisplayAlreadyRunning
	ScheduleResponseStatusInvalidDisplay
	ScheduleResponseStatusUnknown
)

type ScheduleResponseBody struct {
	Status  ScheduleResponseStatus
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
