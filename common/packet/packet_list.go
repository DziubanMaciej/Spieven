package packet

import "math"

type ListRequestBodyFilter struct {
	IdFilter             int
	NameFilter           string
	XorgDisplayFilter    string
	WaylandDisplayFilter string

	HasIdFilter             bool `json:"-"`
	HasNameFilter           bool `json:"-"`
	HasXorgDisplayFilter    bool `json:"-"`
	HasWaylandDisplayFilter bool `json:"-"`
	HasAnyFilter            bool `json:"-"`
}

func (filter *ListRequestBodyFilter) Derive() {
	filter.HasIdFilter = filter.IdFilter != math.MaxInt
	filter.HasNameFilter = filter.NameFilter != ""
	filter.HasXorgDisplayFilter = filter.XorgDisplayFilter != ""
	filter.HasWaylandDisplayFilter = filter.WaylandDisplayFilter != ""
	filter.HasAnyFilter = filter.HasIdFilter || filter.HasNameFilter || filter.HasXorgDisplayFilter || filter.HasWaylandDisplayFilter
}

type ListRequestBody struct {
	Filter             ListRequestBodyFilter
	IncludeDeactivated bool
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
	OutFilePath            string
	MaxSubsequentFailures  int
	UserIndex              int
	IsDeactivated          bool
	DeactivationReason     string
	FriendlyName           string
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
