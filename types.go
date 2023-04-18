package main

import "time"

type metadata struct {
	WindowDuration string `json:"window-duration"`
	RollupTs       int64  `json:"rollup-ts"`
	Uptime         int64  `json:"uptime"`
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
			Uptime:         int64(now.Sub(startedAt).Seconds()),
		},
		Data: map[string]float64{},
	}
}

func (r *rollups) PutData(name string, value float64) {
	r.Data[name] = value
}
