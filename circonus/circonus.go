package circonus

import (
	"errors"
	"time"
    "strings"

	cgm "github.com/circonus-labs/circonus-gometrics"
)

type CirconusSink struct {
	metrics *cgm.CirconusMetrics
}

func NewCirconusSink(
	apiToken string, // required
	submissionUrl string, // optional
	checkId int, // optional
    instanceId string, // optional
    searchTag string, // optional
	apiApp string, // optional "circonus-gometrics"
	apiHost string, // optional "api.circonus.com"
	interval int, // optional (10 seconds)
	brokerId int, // optional (if auto-create check)
) (*CirconusSink, error) {

	// if no submission url and no check id, will automatically create check

	if apiToken == "" {
		return nil, errors.New("API token is required for Circonus sink")
	}

	metrics := cgm.NewCirconusMetrics()

	metrics.ApiToken = apiToken

    if instanceId != "" {
        metrics.InstanceId = instanceId
    }

    if searchTag != "" {
        metrics.SearchTag = searchTag
    }

	if apiApp != "" {
		metrics.ApiApp = apiApp
	}

	if apiHost != "" {
		metrics.ApiHost = apiHost
	}

	if submissionUrl != "" {
		metrics.SubmissionUrl = submissionUrl
	} else if checkId > 0 {
		metrics.CheckId = checkId
	}

	if interval > 0 {
		metrics.Interval = time.Duration(interval) * time.Second
	}

	if brokerId > 0 {
		metrics.BrokerGroupId = brokerId
	}

	return &CirconusSink{
		metrics: metrics,
	}, nil
}


func (s *CirconusSink) Start() {
    s.metrics.Start()
}

func (s *CirconusSink) SetGauge(key []string, val float32) {
	flatKey := s.flattenKey(key)
	s.metrics.SetGauge(flatKey, int64(val))
}

func (s *CirconusSink) EmitKey(key []string, val float32) {
	// NOP
}

func (s *CirconusSink) IncrCounter(key []string, val float32) {
	flatKey := s.flattenKey(key)
	s.metrics.IncrementByValue(flatKey, uint64(val))
}

func (s *CirconusSink) AddSample(key []string, val float32) {
	flatKey := s.flattenKey(key)
	s.metrics.RecordValue(flatKey, float64(val))
}

// Flattens key to metric name
func (s *CirconusSink) flattenKey(parts []string) string {
	joined := strings.Join(parts, "`")
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ':
			return '_'
		default:
			return r
		}
	}, joined)
}
