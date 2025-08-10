package packet

type RefreshBody struct {
	TaskId int
}

func EncodeRefreshPacket(body RefreshBody) (Packet, error) {
	return EncodePacket(PacketIdRefresh, body)
}

func DecodeRefreshPacket(packet Packet) (body RefreshBody, err error) {
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
