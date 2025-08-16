package types

import "math"

type TaskFilter struct {
	IdFilter      int
	AnyNameFilter []string
	DisplayFilter DisplaySelection
	AllTagsFilter []string

	HasIdFilter      bool `json:"-"`
	HasAnyNameFilter bool `json:"-"`
	HasDisplayFilter bool `json:"-"`
	HasAllTagsFilter bool `json:"-"`
	HasAnyFilter     bool `json:"-"`
}

func (filter *TaskFilter) Derive() {
	filter.HasIdFilter = filter.IdFilter != math.MaxInt
	filter.HasAnyNameFilter = len(filter.AnyNameFilter) > 0
	filter.HasDisplayFilter = filter.DisplayFilter.Type != DisplaySelectionTypeNone
	filter.HasAllTagsFilter = len(filter.AllTagsFilter) > 0
	filter.HasAnyFilter = filter.HasIdFilter || filter.HasAnyNameFilter || filter.HasDisplayFilter || filter.HasAllTagsFilter
}
