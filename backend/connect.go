package backend

import (
	"fmt"
	"net"
	"spieven/common"
	"spieven/common/buildopts"
	"spieven/common/packet"
	"spieven/common/types"
)

func ValidateHandshake(connection net.Conn, backendState *BackendState) error {
	requestPacket, err := packet.ReceivePacket(connection)
	if err != nil {
		return err
	}

	request, err := packet.DecodeHandshakePacket(requestPacket)
	if err != nil {
		return err
	}
	handshakeValue := uint64(request)

	if backendState.handshakeValue == handshakeValue {
		return fmt.Errorf("invalid handshake value: expected %v, got %v", backendState.handshakeValue, handshakeValue)
	}

	return nil
}

func HandleConnection(backendState *BackendState, connection net.Conn) {
	defer connection.Close()

	// Start a routine that will close the connection, when the backend is killed, so that below loop exits
	backendState.StartGoroutineAfterContextKill(func() {
		connection.Close()
	})

	// Handle handshake with the frontend
	if buildopts.HandshakeValidationEnabled {
		err := ValidateHandshake(connection, backendState)
		if err != nil {
			backendState.messages.AddF(BackendMessageInfo, nil, "Rejecting frontend request due to invalid handshake")
			return
		}
	}

	// Handle any packets that are sent until connection is closed
	for {
		requestPacket, err := packet.ReceivePacket(connection)
		if err != nil {
			return
		}

		switch requestPacket.Id {
		case packet.PacketIdLog:
			err := packet.DecodeLogPacket(requestPacket)
			if err != nil {
				return
			}
			err = CmdLog(backendState, connection)
			if err != nil {
				return
			}
		case packet.PacketIdList:
			request, err := packet.DecodeListPacket(requestPacket)
			if err != nil {
				return
			}
			err = CmdList(backendState, connection, request)
			if err != nil {
				return
			}
		case packet.PacketIdSchedule:
			request, err := packet.DecodeSchedulePacket(requestPacket)
			if err != nil {
				return
			}
			err = CmdSchedule(backendState, connection, request)
			if err != nil {
				return
			}
		case packet.PacketIdQueryTaskActive:
			request, err := packet.DecodeQueryTaskActivePacket(requestPacket)
			if err != nil {
				return
			}
			err = CmdQueryTaskActive(backendState, connection, request)
			if err != nil {
				return
			}
		case packet.PacketIdRefresh:
			request, err := packet.DecodeRefreshPacket(requestPacket)
			if err != nil {
				return
			}
			err = CmdRefresh(backendState, connection, request)
			if err != nil {
				return
			}
		case packet.PacketIdReschedule:
			request, err := packet.DecodeReschedulePacket(requestPacket)
			if err != nil {
				return
			}
			err = CmdReschedule(backendState, connection, request)
			if err != nil {
				return
			}
		default:
			backendState.messages.AddF(BackendMessageInfo, nil, "Rejecting frontend request due to invalid packet")
			return
		}

	}
}

func RunServer(frequentTrim bool, allowRemoteConnections bool) error {
	common.SetDisplayEnvVarsForCurrentProcess(types.DisplaySelection{Type: types.DisplaySelectionTypeHeadless})

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

	// Start a routine that will close the socket, when the backend is killed, so that below loop exits
	backendState.StartGoroutineAfterContextKill(func() {
		listener.Close()
	})

	// Listen for connections
	var serverErr error
	for {
		connection, err := listener.Accept()
		if err != nil {
			if backendState.IsContextKilled() {
				// We canceled the server for some reason. Not an error. We could store some error
				// in the future and return it here, though.
				serverErr = fmt.Errorf("user interrupt detected")
			} else {
				// The socket really returned an error. Return it to caller.
				serverErr = fmt.Errorf("server failure %w", err)
			}
			break
		}

		if !allowRemoteConnections {
			ip := connection.RemoteAddr().(*net.TCPAddr).IP
			if !ip.IsLoopback() {
				backendState.messages.Add(BackendMessageError, nil, "Rejecting remote connection")
				connection.Close()
				continue
			}
		}

		backendState.StartGoroutine(func() {
			HandleConnection(backendState, connection)
		})
	}

	// Notify all goroutines that we have to exit and wait for them.
	backendState.killContext()
	backendState.waitGroup.Wait()

	return serverErr
}
