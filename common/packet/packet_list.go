package packet

import (
	"math"
	"spieven/common/types"
)

type ListRequestBodyFilter struct {
	IdFilter      int
	NameFilter    string
	DisplayFilter types.DisplaySelection
	AllTagsFilter []string

	HasIdFilter      bool `json:"-"`
	HasNameFilter    bool `json:"-"`
	HasDisplayFilter bool `json:"-"`
	HasAllTagsFilter bool `json:"-"`
	HasAnyFilter     bool `json:"-"`
}

func (filter *ListRequestBodyFilter) Derive() {
	filter.HasIdFilter = filter.IdFilter != math.MaxInt
	filter.HasNameFilter = filter.NameFilter != ""
	filter.HasDisplayFilter = filter.DisplayFilter.Type != types.DisplaySelectionTypeNone
	filter.HasAllTagsFilter = len(filter.AllTagsFilter) > 0
	filter.HasAnyFilter = filter.HasIdFilter || filter.HasNameFilter || filter.HasDisplayFilter || filter.HasAllTagsFilter
}

type ListRequestBody struct {
	Filter                   ListRequestBodyFilter
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
