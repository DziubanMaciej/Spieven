package packet

func EncodeLogPacket() (Packet, error) {
	return EncodePacket(PacketIdLog, nil)
}

func DecodeLogPacket(packet Packet) error {
	return DecodePacket(packet, PacketIdLog, nil)
}

type LogResponseBody []string

func EncodeLogResponsePacket(body LogResponseBody) (Packet, error) {
	return EncodePacket(PacketIdLogResponse, body)
}

func DecodeLogResponsePacket(packet Packet) (result LogResponseBody, err error) {
	err = DecodePacket(packet, PacketIdLogResponse, &result)
	return
}
