package backend

import (
	"fmt"
	"net"
	"supervisor/common"
)

func ValidateHandshake(connection net.Conn) error {
	packet, err := common.ReceivePacket(connection)
	if err != nil {
		return err
	}

	handshake_value, err := common.DecodeHandshakePacket(packet)
	if err != nil {
		return err
	}

	if handshake_value != 123 {
		return fmt.Errorf("invalid handshake value")
	}

	return nil
}

func HandleConnection(backendState *BackendState, connection net.Conn) {
	defer connection.Close()

	if err := ValidateHandshake(connection); err != nil {
		return
	}

	for {
		packet, err := common.ReceivePacket(connection)
		if err != nil {
			return
		}

		switch packet.Id {
		case common.PacketIdSummary:
			err := common.DecodeSummaryPacket(packet)
			if err != nil {
				return
			}
			err = CmdSummary(backendState, connection)
			if err != nil {
				return
			}
		case common.PacketIdList:
			err := common.DecodeListPacket(packet)
			if err != nil {
				return
			}
			err = CmdList(backendState, connection)
			if err != nil {
				return
			}
		case common.PacketIdRegister:
			process_description, err := common.DecodeRegisterPacket(packet)
			if err != nil {
				return
			}
			err = CmdRegister(backendState, connection, process_description)
			if err != nil {
				return
			}
		default:
			return
		}

	}
}

func RunServer() error {
	var backendState BackendState

	// Create socket
	listener, err := net.Listen("tcp4", common.HostWithPort)
	if err != nil {
		return err
	}
	defer listener.Close()

	// Listen for connections
	for {
		connection, err := listener.Accept()
		if err != nil {
			return err
		}

		go HandleConnection(&backendState, connection)
	}
}
