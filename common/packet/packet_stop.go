package packet

import "spieven/common/types"

type StopRequestBody struct {
	TaskId int
}

func EncodeStopPacket(body StopRequestBody) (Packet, error) {
	return EncodePacket(PacketIdStop, body)
}

func DecodeStopPacket(packet Packet) (body StopRequestBody, err error) {
	err = DecodePacket(packet, PacketIdStop, &body)
	return
}

type StopResponseBody struct {
	Status types.StopResponseStatus
}

func EncodeStopResponsePacket(body StopResponseBody) (Packet, error) {
	return EncodePacket(PacketIdStopResponse, body)
}

func DecodeStopResponsePacket(packet Packet) (result StopResponseBody, err error) {
	err = DecodePacket(packet, PacketIdStopResponse, &result)
	return
}
