package backend

import (
	"fmt"
	"net"
	"supervisor/common"
)

func ValidateHandshake(connection net.Conn, backendState *BackendState) error {
	packet, err := common.ReceivePacket(connection)
	if err != nil {
		return err
	}

	handshakeValue, err := common.DecodeHandshakePacket(packet)
	if err != nil {
		return err
	}

	if backendState.handshakeValue == handshakeValue {
		return fmt.Errorf("invalid handshake value: expected %v, got %v", backendState.handshakeValue, handshakeValue)
	}

	return nil
}

func HandleConnection(backendState *BackendState, connection net.Conn) {
	defer connection.Close()

	if common.HandshakeValidationEnabled {
		err := ValidateHandshake(connection, backendState)
		if err != nil {
			backendState.messages.AddF(BackendMessageInfo, nil, "Rejecting frontend request due to invalid handshake")
			return
		}
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
		case common.PacketIdLog:
			err := common.DecodeLogPacket(packet)
			if err != nil {
				return
			}
			err = CmdLog(backendState, connection)
			if err != nil {
				return
			}
		case common.PacketIdList:
			request, err := common.DecodeListPacket(packet)
			if err != nil {
				return
			}
			err = CmdList(backendState, connection, request)
			if err != nil {
				return
			}
		case common.PacketIdSchedule:
			task, err := common.DecodeSchedulePacket(packet)
			if err != nil {
				return
			}
			err = CmdSchedule(backendState, connection, task)
			if err != nil {
				return
			}
		case common.PacketIdQueryTaskActive:
			taskId, err := common.DecodeQueryTaskActivePacket(packet)
			if err != nil {
				return
			}
			err = CmdQueryTaskActive(backendState, connection, taskId)
			if err != nil {
				return
			}
		default:
			backendState.messages.AddF(BackendMessageInfo, nil, "Rejecting frontend request due to invalid packet")
			return
		}

	}
}

func RunServer(frequentTrim bool) error {
	backendState, err := CreateBackendState(frequentTrim)
	if err != nil {
		return err
	}

	// Calculate hash used for verifying frontend requests
	handshakeValue, err := common.CalculateSpievenFileHash()
	if err != nil {
		return nil
	}
	backendState.handshakeValue = handshakeValue

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

		go HandleConnection(backendState, connection)
	}
}
