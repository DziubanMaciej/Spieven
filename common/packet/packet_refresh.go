package packet

import "spieven/common/types"

type RefreshRequestBody struct {
	Filter types.TaskFilter
}

func EncodeRefreshPacket(body RefreshRequestBody) (Packet, error) {
	return EncodePacket(PacketIdRefresh, body)
}

func DecodeRefreshPacket(packet Packet) (body RefreshRequestBody, err error) {
	err = DecodePacket(packet, PacketIdRefresh, &body)
	return
}

type RefreshResponseBody struct {
	RefreshedTasksCount int
	ActiveTasksCount    int
}

func EncodeRefreshResponsePacket(body RefreshResponseBody) (Packet, error) {
	return EncodePacket(PacketIdRefreshResponse, body)
}

func DecodeRefreshResponsePacket(packet Packet) (result RefreshResponseBody, err error) {
	err = DecodePacket(packet, PacketIdRefreshResponse, &result)
	return
}
