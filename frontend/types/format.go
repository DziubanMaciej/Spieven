package frontendtypes

import "fmt"

type ListFormat byte

const (
	ListFormatDefault ListFormat = iota
	ListFormatDetailed
	ListFormatJson
)

func ParseListFormat(value string) (ListFormat, error) {
	switch value {
	case "", "default":
		return ListFormatDefault, nil
	case "detailed":
		return ListFormatDetailed, nil
	case "json":
		return ListFormatJson, nil
	default:
		return ListFormatDefault, fmt.Errorf("invalid format %q, expected one of: default, detailed, json", value)
	}
}


