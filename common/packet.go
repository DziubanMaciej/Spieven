package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

type PacketId byte

const (
	// Frontend->Backend commands
	PacketIdHandshake PacketId = iota
	PacketIdSchedule
	PacketIdList
	PacketIdLog
	PacketIdQueryTaskActive

	// Backend->Frontend commands
	PacketIdScheduleResponse
	PacketIdListResponse
	PacketIdLogResponse
	PacketIdQueryTaskActiveResponse
)

type Packet struct {
	Id     PacketId
	Length uint32
	Data   []byte
}

func EncodePacket(PacketId PacketId, data any) (Packet, error) {
	var result Packet
	var serializedData []byte
	var err error

	if data != nil {
		serializedData, err = json.Marshal(data)
		if err != nil {
			return result, err
		}
	}

	result.Id = PacketId
	result.Data = serializedData
	result.Length = uint32(len(result.Data))

	return result, err
}

func DecodePacket(packet Packet, expectedPacketId PacketId, data any) error {
	if expectedPacketId != packet.Id {
		return fmt.Errorf("invalid PacketId")
	}

	if data != nil {
		return json.Unmarshal(packet.Data, data)
	} else {
		if packet.Length != 0 {
			return fmt.Errorf("unexpected non-zero length for data-less packet")
		}
		return nil
	}
}

func EncodeHandshakePacket(value uint64) (Packet, error) {
	return EncodePacket(PacketIdHandshake, &value)
}

func DecodeHandshakePacket(packet Packet) (result uint64, err error) {
	err = DecodePacket(packet, PacketIdHandshake, &result)
	return
}

type ScheduleBody struct {
	Cmdline       []string
	Cwd           string
	Env           []string
	UserIndex     int
	FriendlyName  string
	CaptureStdout bool
}

func EncodeSchedulePacket(data ScheduleBody) (Packet, error) {
	return EncodePacket(PacketIdSchedule, data)
}

func DecodeSchedulePacket(packet Packet) (result ScheduleBody, err error) {
	err = DecodePacket(packet, PacketIdSchedule, &result)
	return
}

const (
	ScheduleResponseSuccess byte = iota
	ScheduleResponseAlreadyRunning
	ScheduleResponseInvalidDisplay
	ScheduleResponseUnknown
)

type ScheduleResponseBody struct {
	Status  byte
	Id      int
	LogFile string
}

func EncodeScheduleResponsePacket(value ScheduleResponseBody) (Packet, error) {
	return EncodePacket(PacketIdScheduleResponse, &value)
}

func DecodeScheduleResponsePacket(packet Packet) (result ScheduleResponseBody, err error) {
	err = DecodePacket(packet, PacketIdScheduleResponse, &result)
	return
}

type ListFilter struct {
	IdFilter   int
	NameFilter string

	HasIdFilter     bool
	HasNameFilter   bool
	HasUniqueFilter bool `json:"-"`
}

func (filter *ListFilter) Derive() error {
	filter.HasIdFilter = filter.IdFilter != math.MaxInt
	filter.HasNameFilter = filter.NameFilter != ""
	filter.HasUniqueFilter = filter.HasIdFilter || filter.HasNameFilter
	if filter.HasIdFilter && filter.HasNameFilter {
		return errors.New("more than one unique filters found")
	}
	return nil
}

type ListBody struct {
	Filter             ListFilter
	IncludeDeactivated bool
}

func EncodeListPacket(body ListBody) (Packet, error) {
	return EncodePacket(PacketIdList, body)
}

func DecodeListPacket(packet Packet) (body ListBody, err error) {
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

func EncodeQueryTaskActivePacket(body int) (Packet, error) {
	return EncodePacket(PacketIdQueryTaskActive, body)
}

func DecodeQueryTaskActivePacket(packet Packet) (result int, err error) {
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
