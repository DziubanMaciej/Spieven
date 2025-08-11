package types

type ScheduleResponseStatus byte

const (
	ScheduleResponseStatusSuccess ScheduleResponseStatus = iota
	ScheduleResponseStatusAlreadyRunning
	ScheduleResponseStatusNameDisplayAlreadyRunning
	ScheduleResponseStatusInvalidDisplay
	ScheduleResponseStatusTaskNotFound       // only for reschedule
	ScheduleResponseStatusTaskNotDeactivated // only for reschedule
	ScheduleResponseStatusUnknown
)
