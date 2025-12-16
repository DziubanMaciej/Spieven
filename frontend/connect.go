package frontend

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"spieven/common"
	"spieven/common/buildopts"
	"spieven/common/packet"
	"strconv"
	"syscall"
	"time"
)

func ConnectToBackend(allowAutorun bool, serverAddress string, serverPort int) (net.Conn, error) {
	var isCustomAddress bool = true

	// Validate server port
	if serverPort == 0 {
		isCustomAddress = false

		portStr := buildopts.DefaultPort
		var err error
		serverPort, err = strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid default port: %w", err)
		}
	}

	// Validate server address
	if serverAddress == "" {
		isCustomAddress = false

		serverAddress = "localhost"
	}

	// Build the address struct
	hostWithPort := fmt.Sprintf("%s:%d", serverAddress, serverPort)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", hostWithPort)
	if err != nil {
		return nil, err
	}

	connection, err := net.DialTCP("tcp4", nil, tcpAddr)
	if err != nil {
		if buildopts.AutorunBackend && allowAutorun && !isCustomAddress {
			spievenBinary := os.Args[0]
			cmd := exec.Command(spievenBinary, "serve")
			cmd.Stdin = nil
			cmd.Stdout = nil
			cmd.Stderr = nil
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Setsid: true,
			}
			err := cmd.Start()
			if err != nil {
				return nil, errors.New("cannot start Spieven backend")
			}

			dial := func() (*net.TCPConn, error) {
				return net.DialTCP("tcp4", nil, tcpAddr)
			}
			connection, err = common.TryCallWithTimeouts(dial, time.Millisecond*1300, 13)
			if err != nil {
				return nil, errors.New("cannot connect to Spieven backend even after starting it in background")
			}
		} else {
			return nil, errors.New("cannot connect to Spieven backend (didn't try to start it)")
		}
	}

	if buildopts.HandshakeValidationEnabled {
		handshakeValue, err := common.CalculateSpievenFileHash()
		if err != nil {
			connection.Close()
			return nil, err
		}

		requestPacket, err := packet.EncodeHandshakePacket(packet.HandshakeRequestBody(handshakeValue))
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
