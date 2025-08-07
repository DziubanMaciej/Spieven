package frontend

import (
	"net"
	"supervisor/common"
	"supervisor/common/packet"
)

func ConnectToBackend() (net.Conn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", common.HostWithPort)
	if err != nil {
		return nil, err
	}

	connection, err := net.DialTCP("tcp4", nil, tcpAddr)
	if err != nil {
		// TODO start the backend and disown it
		return nil, err
	}

	if common.HandshakeValidationEnabled {
		handshakeValue, err := common.CalculateSpievenFileHash()
		if err != nil {
			return nil, err
		}

		requestPacket, err := packet.EncodeHandshakePacket(handshakeValue)
		if err != nil {
			connection.Close()
			return nil, err
		}

		err = packet.SendPacket(connection, requestPacket)
		if err != nil {
			connection.Close()
			return nil, err
		}
	}

	return connection, nil
}
