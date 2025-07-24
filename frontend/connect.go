package frontend

import (
	"net"
	"supervisor/common"
)

func ConnectToBackend() (net.Conn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", common.HostWithPort)
	if err != nil {
		return nil, err
	}

	connection, err := net.DialTCP("tcp4", nil, tcpAddr)
	if err != nil {
		return nil, err
	}

	packet, err := common.EncodeHandshakePacket(123)
	if err != nil {
		connection.Close()
		return nil, err
	}

	err = common.SendPacket(connection, packet)
	if err != nil {
		connection.Close()
		return nil, err
	}

	return connection, nil
}
