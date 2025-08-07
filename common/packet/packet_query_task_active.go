package packet

type QueryTaskActiveRequestBody int

func EncodeQueryTaskActivePacket(body QueryTaskActiveRequestBody) (Packet, error) {
	return EncodePacket(PacketIdQueryTaskActive, body)
}

func DecodeQueryTaskActivePacket(packet Packet) (result QueryTaskActiveRequestBody, err error) {
	err = DecodePacket(packet, PacketIdQueryTaskActive, &result)
	return
}

type QueryTaskActiveResponseBody byte

const (
	QueryTaskActiveResponseBodyActive QueryTaskActiveResponseBody = iota
	QueryTaskActiveResponseBodyInactive
	QueryTaskActiveResponseInvalidTask
)

func EncodeQueryTaskActiveResponsePacket(body QueryTaskActiveResponseBody) (Packet, error) {
	return EncodePacket(PacketIdQueryTaskActiveResponse, body)
}

func DecodeQueryTaskActiveResponsePacket(packet Packet) (result QueryTaskActiveResponseBody, err error) {
	err = DecodePacket(packet, PacketIdQueryTaskActiveResponse, &result)
	return
}
