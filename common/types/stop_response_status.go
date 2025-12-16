package types

type StopResponseStatus byte

const (
	StopResponseStatusSuccess StopResponseStatus = iota
	StopResponseStatusTaskNotFound
	StopResponseStatusAlreadyStopped
	StopResponseStatusUnknown
)

