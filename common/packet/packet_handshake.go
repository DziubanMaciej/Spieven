package packet

type HandshakeRequestBody uint64

func EncodeHandshakePacket(value HandshakeRequestBody) (Packet, error) {
	return EncodePacket(PacketIdHandshake, &value)
}

func DecodeHandshakePacket(packet Packet) (result HandshakeRequestBody, err error) {
	err = DecodePacket(packet, PacketIdHandshake, &result)
	return
}
