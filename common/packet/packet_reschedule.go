package packet

type RescheduleRequestBody struct {
	TaskId int
}

func EncodeReschedulePacket(body RescheduleRequestBody) (Packet, error) {
	return EncodePacket(PacketIdReschedule, body)
}

func DecodeReschedulePacket(packet Packet) (body RescheduleRequestBody, err error) {
	err = DecodePacket(packet, PacketIdReschedule, &body)
	return
}

type RescheduleResponseBody ScheduleResponseBody

func EncodeRescheduleResponsePacket(body RescheduleResponseBody) (Packet, error) {
	return EncodePacket(PacketIdRescheduleResponse, body)
}

func DecodeRescheduleResponsePacket(packet Packet) (result RescheduleResponseBody, err error) {
	err = DecodePacket(packet, PacketIdRescheduleResponse, &result)
	return
}
