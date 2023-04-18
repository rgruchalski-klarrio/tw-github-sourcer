package types

import (
	"fmt"
	"strings"
	"time"

	"github.com/Klarrio/tw-github-sourcer/defaults"
)

type Metadata struct {
	WindowDuration string `json:"window-duration"`
	RollupTs       int64  `json:"rollup-ts"`
	Uptime         string `json:"uptime"`
}

type Rollups struct {
	Metadata *Metadata          `json:"metadata"`
	Data     map[string]float64 `json:"data"`
}

func NewRollups() *Rollups {
	now := time.Now()
	return &Rollups{
		Metadata: &Metadata{
			WindowDuration: defaults.SlidingWindowDuration.String(),
			RollupTs:       now.Unix(),
			Uptime:         now.Sub(defaults.ProgramStartedAt).String(),
		},
		Data: map[string]float64{},
	}
}

func (r *Rollups) PutData(name string, value float64) {
	r.Data[strings.TrimPrefix(name, fmt.Sprintf("%s_%s_", defaults.PrometheusNamespace, defaults.PrometheusSubsystem))] = value
}
