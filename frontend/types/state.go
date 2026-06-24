package frontendtypes

import "fmt"

type TaskStatusFilter byte

const (
	TaskStatusFilterActive TaskStatusFilter = iota
	TaskStatusFilterInactive
	TaskStatusFilterAll
)

const TaskStatusFilterStrValues = "all, active, inactive"

func ParseTaskStatusFilter(value string) (TaskStatusFilter, error) {
	switch value {
	case "", "all":
		return TaskStatusFilterAll, nil
	case "active":
		return TaskStatusFilterActive, nil
	case "inactive":
		return TaskStatusFilterInactive, nil
	default:
		return TaskStatusFilterAll, fmt.Errorf("invalid task status filter %q, expected one of: %v", value, TaskStatusFilterStrValues)
	}
}
