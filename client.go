// Simple client to the [Datadog API](http://docs.datadoghq.com/api/).
package datadog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	ENDPOINT = "https://app.datadoghq.com/api/v1"
)

type Client struct {
	Host   string
	ApiKey string
}

type Event struct {
	Title     string   `json:"title"`
	Text      string   `json:"text"`
	Timestamp int64    `json:"date_happened,omitempty"`
	Host      string   `json:"host,omitempty"`
	Tags      []string `json:"tags,omitempty"`

	// Event priority can be "normal" or "low", defaults to "normal"
	Priority string `json:"priority,omitempty"`
	// Event type can be "error", "warning", "info" or "success", defaults to "into"
	Type string `json:"alert_type,omitempty"`
	// An arbitrary string to use for aggregation, max length of 100 characters.
	Key string `json:"aggregation_key,omitempty"`
	// The type of event being posted. Options: nagios, hudson, jenkins, user, my apps, feed, chef, puppet, git, bitbucket, fabric, capistrano
	Source string `json:"source_type_name,omitempty"`
}

// New creates a new Datadog client. In EC2, datadog expects the hostname to be the
// instance ID rather than `gethostname(2)`. However, that value can be obtained
// with `os.Hostname()`.
func New(host, apiKey string) *Client {
	return &Client{
		Host:   host,
		ApiKey: apiKey,
	}
}

// SeriesUrl gets an authenticated URL to POST series data to. In Datadog's examples, this
// value is 'https://app.datadoghq.com/api/v1/series?api_key=9775a026f1ca7d1...'
func (c *Client) SeriesUrl() string {
	return ENDPOINT + "/series?api_key=" + c.ApiKey
}

// EventsUrl gets an authenticated URL to POST series data to. In Datadog's examples, this
// value is 'https://app.datadoghq.com/api/v1/events?api_key=9775a026f1ca7d1...'
func (c *Client) EventsUrl() string {
	return ENDPOINT + "/events?api_key=" + c.ApiKey
}

// PostSeries posts an array of series data to the Datadog API. The API expects an object,
// not an array, so it will be wrapped in a `seriesMessage` with a single
// `series` field.
func (c *Client) PostSeries(series []*Series) error {
	return c.post(c.SeriesUrl(), &seriesMessage{series})
}

// PostEvent post a single event to the Datadog API.
func (c *Client) PostEvent(event *Event) (err error) {
	if event.Host == "" {
		event.Host = c.Host
	}
	return c.post(c.EventsUrl(), event)
}

// Reporter creates a `MetricReporter`. The returned
// reporter will not be started.
func (c *Client) Reporter(tags ...string) *MetricReporter {
	return NewReporter(c, tags...)
}

// Private marshal
func (c *Client) marshal(v interface{}) (io.Reader, error) {
	body := bytes.Buffer{}
	if err := json.NewEncoder(&body).Encode(v); err != nil {
		return nil, err
	}
	return &body, nil
}

// Private HTTP post
func (c *Client) post(url string, v interface{}) error {
	body, err := c.marshal(v)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		return fmt.Errorf("Bad Datadog response: '%s'", resp.Status)
	}
	return nil
}
