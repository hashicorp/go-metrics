// Circonus Metrics Sink

package circonus

import (
	"strings"
	"time"

	cgm "github.com/circonus-labs/circonus-gometrics"
)

// Circonus Sink provides an interface to forward metrics to Circonus with
// automatic check creation and metric management
type CirconusSink struct {
	metrics *cgm.CirconusMetrics
}

// Circonus Sink configuration options
//   - If no ApiToken, CheckSubmissionUrl, or CheckId are supplied an error will be returned.
//   - If no ApiToken is supplied check management is disabled.
type Config struct {
	ApiToken           string        // optional (eliding turns off auto-create check and check management)
	ApiApp             string        // optional "circonus-gometrics"
	ApiUrl             string        // optional "https://api.circonus.com/v2"
	SubmitInterval     time.Duration // optional (10 seconds)
	CheckSubmissionUrl string        // optional
	CheckId            int           // optional
	CheckInstanceId    string        // optional "hostname:app_name /cgm"
	CheckSearchTag     string        // optional "service:app_name"
	BrokerId           int           // optional (used for auto-create check if token supplied)
	BrokerSelectTag    string        // optional tag to use to select broker (if auto-create check)
}

// Create new CirconusSink
//
// one of the following must be supplied:
//    - ApiToken - search for an existing check or create a new check
//    - ApiToken + CheckId - the check identified by check id will be used
//    - ApiToken + CheckSubmissionUrl - the check identified by the submission url will be used
//    - CheckSubmissionUrl - the check identified by the submission url will be used
//      metric management will be *disabled*
//
// Note: If submission url is supplied w/o an api token, the public circonus ca cert will be used
// to verify the broker for metrics submission.
func NewCirconusSink(cc *Config) (*CirconusSink, error) {

	cfg := &cgm.Config{}

	if cc != nil {
		cfg.CheckManager.Api.Token.Key = cc.ApiToken
		cfg.CheckManager.Api.Token.App = cc.ApiApp
		cfg.CheckManager.Api.Url = cc.ApiUrl
		cfg.CheckManager.Check.InstanceId = cc.CheckInstanceId
		cfg.CheckManager.Check.SearchTag = cc.CheckSearchTag
		cfg.CheckManager.Check.SubmissionUrl = cc.CheckSubmissionUrl
		cfg.CheckManager.Check.Id = cc.CheckId
		cfg.CheckManager.Broker.Id = cc.BrokerId
		cfg.CheckManager.Broker.SelectTag = cc.BrokerSelectTag
		cfg.Interval = cc.SubmitInterval
	}

	metrics, err := cgm.NewCirconusMetrics(cfg)
	if err != nil {
		return nil, err
	}

	return &CirconusSink{
		metrics: metrics,
	}, nil
}

// Start submitting metrics to Circonus (flush every SubmitInterval)
func (s *CirconusSink) Start() {
	s.metrics.Start()
}

// Manually flush metrics to Circonus
func (s *CirconusSink) Flush() {
	s.metrics.Flush()
}

// Set value for a gauge metric
func (s *CirconusSink) SetGauge(key []string, val float32) {
	flatKey := s.flattenKey(key)
	s.metrics.SetGauge(flatKey, int64(val))
}

// not implemented in circonus
func (s *CirconusSink) EmitKey(key []string, val float32) {
	// NOP
}

// Increment a counter metric
func (s *CirconusSink) IncrCounter(key []string, val float32) {
	flatKey := s.flattenKey(key)
	s.metrics.IncrementByValue(flatKey, uint64(val))
}

// Add a sample to a histogram metric
func (s *CirconusSink) AddSample(key []string, val float32) {
	flatKey := s.flattenKey(key)
	s.metrics.RecordValue(flatKey, float64(val))
}

// Flattens key to Circonus metric name
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
