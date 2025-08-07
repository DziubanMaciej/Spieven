package packet

import (
	"encoding/binary"
	"io"
)

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
