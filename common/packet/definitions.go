package packet

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

func SendPacket(writer io.Writer, packet Packet) error {
	var bytes []byte
	bytes = append(bytes, byte(packet.Id))
	bytes = binary.LittleEndian.AppendUint32(bytes, packet.Length)
	bytes = append(bytes, packet.Data...)

	written := 0
	for written < len(bytes) {
		writtenThisIteration, err := writer.Write(bytes[written:])
		if err != nil {
			return err
		}

		written += writtenThisIteration
	}
	return nil
}

func ReceivePacket(reader io.Reader) (Packet, error) {
	readBytes := func(bytes []byte) error {
		bytesRead := 0
		for bytesRead < len(bytes) {
			bytesReadThisIteration, err := reader.Read(bytes[bytesRead:])
			if err != nil {
				return err
			}

			bytesRead += bytesReadThisIteration
		}
		return nil
	}

	var receiveBuffer [4]byte
	var result Packet

	if err := readBytes(receiveBuffer[:1]); err != nil {
		return result, err
	}
	result.Id = PacketId(receiveBuffer[0])

	if err := readBytes(receiveBuffer[:4]); err != nil {
		return result, err
	}
	result.Length = binary.LittleEndian.Uint32(receiveBuffer[:4])
	result.Data = make([]byte, result.Length)

	if err := readBytes(result.Data); err != nil {
		return result, err
	}

	return result, nil
}

type DisplaySelection struct {
	Type DisplaySelectionType
	Name string
}

func (selection *DisplaySelection) ParseDisplaySelection(val string) error {
	if val == "" {
		selection.Type = DisplaySelectionTypeNone
		selection.Name = ""
		return nil
	}

	switch val[0] {
	case 'h':
		selection.Type = DisplaySelectionTypeHeadless
		selection.Name = ""
		return nil
	case 'x':
		selection.Type = DisplaySelectionTypeXorg
	case 'w':
		selection.Type = DisplaySelectionTypeWayland
	default:
		return errors.New("invalid display selection - it must be either headless, xorg or wayland")
	}

	selection.Name = val[1:]
	if selection.Name == "" {
		return errors.New("invalid display selection - it must contain display name")
	}
	return nil
}

const DisplaySelectionHelpString = "Use \"h\" for headless, \"x$DISPLAY\" for xorg or \"w$WAYLAND_DISPLAY\" for wayland."
