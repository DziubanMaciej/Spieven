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

	// Backend->Frontend commands
	PacketIdSummaryResponse
	PacketIdRegisterResponse
)

type Packet struct {
	Id     PacketId
	Length uint32
	Data   []byte
}

func EncodePacket(PacketId PacketId, data any) (Packet, error) {
	var result Packet
	var serialized_command []byte
	var err error

	if data != nil {
		serialized_command, err = json.Marshal(data)
		if err != nil {
			return result, err
		}
	}

	result.Id = PacketId
	result.Data = []byte(serialized_command)
	result.Length = uint32(len(result.Data))

	return result, err
}

func DecodePacket(packet Packet, expected_command_id PacketId, data any) error {
	if expected_command_id != packet.Id {
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

func EncodeHandshakePacket(value int) (Packet, error) {
	return EncodePacket(PacketIdHandshake, &value)
}

func DecodeHandshakePacket(packet Packet) (result int, err error) {
	err = DecodePacket(packet, PacketIdHandshake, &result)
	return
}

type RegisterBody struct {
	Cmdline []string
	Cwd     string
}

func EncodeRegisterPacket(processDescription RegisterBody) (Packet, error) {
	return EncodePacket(PacketIdRegister, processDescription)
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
