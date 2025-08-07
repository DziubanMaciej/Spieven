package packet

type ScheduleBody struct {
	Cmdline       []string
	Cwd           string
	Env           []string
	UserIndex     int
	FriendlyName  string
	CaptureStdout bool
}

func EncodeSchedulePacket(data ScheduleBody) (Packet, error) {
	return EncodePacket(PacketIdSchedule, data)
}

func DecodeSchedulePacket(packet Packet) (result ScheduleBody, err error) {
	err = DecodePacket(packet, PacketIdSchedule, &result)
	return
}

const (
	ScheduleResponseSuccess byte = iota
	ScheduleResponseAlreadyRunning
	ScheduleResponseNameDisplayAlreadyRunning
	ScheduleResponseInvalidDisplay
	ScheduleResponseUnknown
)

type ScheduleResponseBody struct {
	Status  byte
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
