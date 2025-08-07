package packet

func EncodeHandshakePacket(value uint64) (Packet, error) {
	return EncodePacket(PacketIdHandshake, &value)
}

func DecodeHandshakePacket(packet Packet) (result uint64, err error) {
	err = DecodePacket(packet, PacketIdHandshake, &result)
	return
}
