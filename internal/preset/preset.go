package preset

import (
	"fmt"
	"sort"
	"strings"
)

type Definition struct {
	Name   string
	Width  int
	Height int
	Hz     int
}

var definitions = map[string]Definition{
	"1080p": {
		Name:   "1080p",
		Width:  1920,
		Height: 1080,
		Hz:     60,
	},
	"1200p": {
		Name:   "1200p",
		Width:  1920,
		Height: 1200,
		Hz:     60,
	},
	"1440p": {
		Name:   "1440p",
		Width:  2560,
		Height: 1440,
		Hz:     60,
	},
	"4k": {
		Name:   "4k",
		Width:  3840,
		Height: 2160,
		Hz:     60,
	},
	"3k": {
		Name:   "3k",
		Width:  3200,
		Height: 1800,
		Hz:     60,
	},
	"2.8k": {
		Name:   "2.8k",
		Width:  2880,
		Height: 1800,
		Hz:     60,
	},
	"1600p": {
		Name:   "1600p",
		Width:  2560,
		Height: 1600,
		Hz:     60,
	},
	"uwqhd": {
		Name:   "uwqhd",
		Width:  3440,
		Height: 1440,
		Hz:     60,
	},
	"uw1600p": {
		Name:   "uw1600p",
		Width:  3840,
		Height: 1600,
		Hz:     60,
	},
	"dual-1080p": {
		Name:   "dual-1080p",
		Width:  3840,
		Height: 1080,
		Hz:     60,
	},
	"3:2-large": {
		Name:   "3:2-large",
		Width:  2496,
		Height: 1664,
		Hz:     60,
	},
	"3:2-medium": {
		Name:   "3:2-medium",
		Width:  2256,
		Height: 1504,
		Hz:     60,
	},
	"surface-pro": {
		Name:   "surface-pro",
		Width:  2736,
		Height: 1824,
		Hz:     60,
	},
}

func Parse(name string) (Definition, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return Definition{}, nil
	}

	definition, ok := definitions[key]
	if !ok {
		return Definition{}, fmt.Errorf("unsupported preset %q: want one of %s", name, strings.Join(Names(), ", "))
	}

	return definition, nil
}

func Names() []string {
	names := make([]string, 0, len(definitions))
	for name := range definitions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
