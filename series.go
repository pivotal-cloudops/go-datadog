package datadog

type seriesMessage struct {
	Series []*Series `json:"series,omitempty"`
}

type Series struct {
	Metric string           `json:"metric"`
	Points [][2]interface{} `json:"points"`
	Type   string           `json:"type"`
	Host   string           `json:"host,omitempty"`
	Tags   []string         `json:"tags,omitempty"`
}

// NewSeries builds a series
func NewSeries(name string, t int64, v interface{}, tags []string, mt string) *Series {
	return &Series{
		Metric: name,
		Points: [][2]interface{}{[2]interface{}{t, v}},
		Type:   mt,
		Tags:   tags,
	}
}
