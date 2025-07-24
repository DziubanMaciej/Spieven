package frontend

import (
	"fmt"
	"net"
	"supervisor/common"
)

func CmdSummary(backendConnection net.Conn) error {
	request_packet, err := common.EncodeSummaryPacket()
	if err != nil {
		return err
	}

	err = common.SendPacket(backendConnection, request_packet)
	if err != nil {
		return err
	}

	response_packet, err := common.ReceivePacket(backendConnection)
	if err != nil {
		return err
	}

	summary, err := common.DecodeSummaryResponsePacket(response_packet)
	if err != nil {
		return err
	}

	fmt.Printf("Version: %v\n", summary.Version)
	fmt.Printf("Running processes: %v\n", summary.ConnectionCount)
	return nil
}

func CmdRegister(backendConnection net.Conn, args []string) error {
	body := common.RegisterBody{
		Cmdline: args,
		Cwd:     "",
	}

	packet, err := common.EncodeRegisterPacket(body)
	if err != nil {
		return err
	}

	return common.SendPacket(backendConnection, packet)
}
