package packet

import "spieven/common/types"

type ListRequestBody struct {
	Filter                   types.TaskFilter
	IncludeDeactivated       bool
	IncludeDeactivatedAlways bool
	UniqueNames              bool
}

func EncodeListPacket(body ListRequestBody) (Packet, error) {
	return EncodePacket(PacketIdList, body)
}

func DecodeListPacket(packet Packet) (body ListRequestBody, err error) {
	err = DecodePacket(packet, PacketIdList, &body)
	return
}

type ListResponseBodyItem struct {
	Id                     int
	Cmdline                []string
	Cwd                    string
	Display                types.DisplaySelection
	OutFilePath            string
	MaxSubsequentFailures  int
	IsDeactivated          bool
	DeactivationReason     string
	FriendlyName           string
	Tags                   []string
	RunCount               int
	FailureCount           int
	SubsequentFailureCount int
	LastExitValue          int
	LastStdout             string
	HasLastStdout          bool
}
type ListResponseBody []ListResponseBodyItem

func EncodeListResponsePacket(body ListResponseBody) (Packet, error) {
	return EncodePacket(PacketIdListResponse, body)
}

func DecodeListResponsePacket(packet Packet) (result ListResponseBody, err error) {
	err = DecodePacket(packet, PacketIdListResponse, &result)
	return
}
