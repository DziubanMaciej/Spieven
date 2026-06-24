package packet

type ResumeRequestBody struct {
	TaskId int
}

func EncodeResumePacket(body ResumeRequestBody) (Packet, error) {
	return EncodePacket(PacketIdResume, body)
}

func DecodeResumePacket(packet Packet) (body ResumeRequestBody, err error) {
	err = DecodePacket(packet, PacketIdResume, &body)
	return
}

type ResumeResponseBody RunResponseBody

func EncodeResumeResponsePacket(body ResumeResponseBody) (Packet, error) {
	return EncodePacket(PacketIdResumeResponse, body)
}

func DecodeResumeResponsePacket(packet Packet) (result ResumeResponseBody, err error) {
	err = DecodePacket(packet, PacketIdResumeResponse, &result)
	return
}
