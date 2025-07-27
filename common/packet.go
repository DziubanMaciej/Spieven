package common

import (
	"encoding/json"
	"fmt"
)

type PacketId byte

const (
	// Frontend->Backend commands
	PacketIdHandshake PacketId = iota
	PacketIdRegister
	PacketIdSummary
	PacketIdList
	PacketIdLog

	// Backend->Frontend commands
	PacketIdSummaryResponse
	PacketIdRegisterResponse
	PacketIdListResponse
	PacketIdLogResponse
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

type RegisterBody struct {
	Cmdline   []string
	Cwd       string
	Env       []string
	UserIndex int
}

func EncodeRegisterPacket(data RegisterBody) (Packet, error) {
	return EncodePacket(PacketIdRegister, data)
}

func DecodeRegisterPacket(packet Packet) (result RegisterBody, err error) {
	err = DecodePacket(packet, PacketIdRegister, &result)
	return
}

func EncodeSummaryPacket() (Packet, error) {
	return EncodePacket(PacketIdSummary, nil)
}

func DecodeSummaryPacket(packet Packet) error {
	return DecodePacket(packet, PacketIdSummary, nil)
}

type SummaryResponseBody struct {
	Version         string
	ConnectionCount int
}

func EncodeSummaryResponsePacket(data SummaryResponseBody) (Packet, error) {
	return EncodePacket(PacketIdRegister, data)
}

func DecodeSummaryResponsePacket(packet Packet) (result SummaryResponseBody, err error) {
	err = DecodePacket(packet, PacketIdRegister, &result)
	return
}

func EncodeRegisterResponsePacket(value bool) (Packet, error) {
	return EncodePacket(PacketIdRegisterResponse, &value)
}

func DecodeRegisterResponsePacket(packet Packet) (result bool, err error) {
	err = DecodePacket(packet, PacketIdRegisterResponse, &result)
	return
}

func EncodeListPacket() (Packet, error) {
	return EncodePacket(PacketIdList, nil)
}

func DecodeListPacket(packet Packet) error {
	return DecodePacket(packet, PacketIdList, nil)
}

type ListResponseBody []struct {
	Id                    int
	Cmdline               []string
	Cwd                   string
	OutFilePath           string
	MaxSubsequentFailures int
	UserIndex             int
	IsDeactivated         bool
	DeactivationReason    string
}

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
