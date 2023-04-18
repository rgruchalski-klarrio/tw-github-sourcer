package main

import (
	"fmt"
	"strings"
	"time"
)

type metadata struct {
	WindowDuration string `json:"window-duration"`
	RollupTs       int64  `json:"rollup-ts"`
	Uptime         string `json:"uptime"`
}

type rollups struct {
	Metadata *metadata          `json:"metadata"`
	Data     map[string]float64 `json:"data"`
}

func newRollups(startedAt time.Time, duration time.Duration) *rollups {
	now := time.Now()
	return &rollups{
		Metadata: &metadata{
			WindowDuration: duration.String(),
			RollupTs:       now.Unix(),
			Uptime:         now.Sub(startedAt).String(),
		},
		Data: map[string]float64{},
	}
}

func (r *rollups) PutData(name string, value float64) {
	r.Data[strings.TrimPrefix(name, fmt.Sprintf("%s_%s_", prometheusNamespace, prometheusSubsystem))] = value
}
