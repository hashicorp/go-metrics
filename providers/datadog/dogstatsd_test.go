package datadog

import (
	"net"
	"reflect"
	"testing"

	"github.com/hugoluchessi/go-metrics"
)

var EmptyTags []metrics.Label

const (
	DogStatsdAddr    = "127.0.0.1:7254"
	HostnameEnabled  = true
	HostnameDisabled = false
	TestHostname     = "test_hostname"
)

func MockGetHostname() string {
	return TestHostname
}

var ParseKeyTests = []struct {
	KeyToParse        []string
	Tags              []metrics.Label
	PropagateHostname bool
	ExpectedKey       []string
	ExpectedTags      []metrics.Label
}{
	{[]string{"a", MockGetHostname(), "b", "c"}, EmptyTags, HostnameDisabled, []string{"a", "b", "c"}, EmptyTags},
	{[]string{"a", "b", "c"}, EmptyTags, HostnameDisabled, []string{"a", "b", "c"}, EmptyTags},
	{[]string{"a", "b", "c"}, EmptyTags, HostnameEnabled, []string{"a", "b", "c"}, []metrics.Label{{Name: "host", Value: MockGetHostname()}}},
}

var FlattenKeyTests = []struct {
	KeyToFlatten []string
	Expected     string
}{
	{[]string{"a", "b", "c"}, "a.b.c"},
	{[]string{"spaces must", "flatten", "to", "underscores"}, "spaces_must.flatten.to.underscores"},
}

var MetricSinkTests = []struct {
	Method            string
	Metric            []string
	Value             interface{}
	Tags              []metrics.Label
	PropagateHostname bool
	Expected          string
}{
	{"SetGauge", []string{"foo", "bar"}, float32(42), EmptyTags, HostnameDisabled, "foo.bar:42.000000|g"},
	{"SetGauge", []string{"foo", "bar", "baz"}, float32(42), EmptyTags, HostnameDisabled, "foo.bar.baz:42.000000|g"},
	{"AddSample", []string{"sample", "thing"}, float32(4), EmptyTags, HostnameDisabled, "sample.thing:4.000000|ms"},
	{"IncrCounter", []string{"count", "me"}, float32(3), EmptyTags, HostnameDisabled, "count.me:3|c"},

	{"SetGauge", []string{"foo", "baz"}, float32(42), []metrics.Label{{Name: "my_tag", Value: ""}}, HostnameDisabled, "foo.baz:42.000000|g|#my_tag"},
	{"SetGauge", []string{"foo", "baz"}, float32(42), []metrics.Label{{Name: "my tag", Value: "my_value"}}, HostnameDisabled, "foo.baz:42.000000|g|#my_tag:my_value"},
	{"SetGauge", []string{"foo", "bar"}, float32(42), []metrics.Label{{Name: "my_tag", Value: "my_value"}, {Name: "other_tag", Value: "other_value"}}, HostnameDisabled, "foo.bar:42.000000|g|#my_tag:my_value,other_tag:other_value"},
	{"SetGauge", []string{"foo", "bar"}, float32(42), []metrics.Label{{Name: "my_tag", Value: "my_value"}, {Name: "other_tag", Value: "other_value"}}, HostnameEnabled, "foo.bar:42.000000|g|#my_tag:my_value,other_tag:other_value,host:test_hostname"},
}

func mockNewSink(addr string, labels []metrics.Label, tagWithHostname bool) *Sink {
	dog, _ := NewSink(addr, MockGetHostname())
	_, tags := dog.getFlatkeyAndCombinedLabels(nil, labels)
	dog.SetTags(tags)
	if tagWithHostname {
		dog.EnableHostNamePropagation()
	}

	return dog
}

func setupTestServerAndBuffer(t *testing.T) (*net.UDPConn, []byte) {
	udpAddr, err := net.ResolveUDPAddr("udp", DogStatsdAddr)
	if err != nil {
		t.Fatal(err)
	}
	server, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatal(err)
	}
	return server, make([]byte, 1024)
}

func TestParseKey(t *testing.T) {
	for _, tt := range ParseKeyTests {
		dog := mockNewSink(DogStatsdAddr, tt.Tags, tt.PropagateHostname)
		key, tags := dog.parseKey(tt.KeyToParse)

		if !reflect.DeepEqual(key, tt.ExpectedKey) {
			t.Fatalf("Key Parsing failed for %v", tt.KeyToParse)
		}

		if !reflect.DeepEqual(tags, tt.ExpectedTags) {
			t.Fatalf("Tag Parsing Failed for %v, %v != %v", tt.KeyToParse, tags, tt.ExpectedTags)
		}
	}
}

func TestFlattenKey(t *testing.T) {
	dog := mockNewSink(DogStatsdAddr, EmptyTags, HostnameDisabled)
	for _, tt := range FlattenKeyTests {
		if !reflect.DeepEqual(dog.flattenKey(tt.KeyToFlatten), tt.Expected) {
			t.Fatalf("Flattening %v failed", tt.KeyToFlatten)
		}
	}
}

func TestMetricSink(t *testing.T) {
	server, buf := setupTestServerAndBuffer(t)
	defer server.Close()

	for _, tt := range MetricSinkTests {
		dog := mockNewSink(DogStatsdAddr, tt.Tags, tt.PropagateHostname)
		method := reflect.ValueOf(dog).MethodByName(tt.Method)
		method.Call([]reflect.Value{
			reflect.ValueOf(tt.Metric),
			reflect.ValueOf(tt.Value)})
		assertServerMatchesExpected(t, server, buf, tt.Expected)
	}
}

func TestTaggableMetrics(t *testing.T) {
	server, buf := setupTestServerAndBuffer(t)
	defer server.Close()

	dog := mockNewSink(DogStatsdAddr, EmptyTags, HostnameDisabled)

	dog.AddSampleWithLabels([]string{"sample", "thing"}, float32(4), []metrics.Label{{Name: "tagkey", Value: "tagvalue"}})
	assertServerMatchesExpected(t, server, buf, "sample.thing:4.000000|ms|#tagkey:tagvalue")

	dog.SetGaugeWithLabels([]string{"sample", "thing"}, float32(4), []metrics.Label{{Name: "tagkey", Value: "tagvalue"}})
	assertServerMatchesExpected(t, server, buf, "sample.thing:4.000000|g|#tagkey:tagvalue")

	dog.IncrCounterWithLabels([]string{"sample", "thing"}, float32(4), []metrics.Label{{Name: "tagkey", Value: "tagvalue"}})
	assertServerMatchesExpected(t, server, buf, "sample.thing:4|c|#tagkey:tagvalue")

	dog = mockNewSink(DogStatsdAddr, []metrics.Label{{Name: "global"}}, HostnameEnabled) // with hostname, global tags
	dog.IncrCounterWithLabels([]string{"sample", "thing"}, float32(4), []metrics.Label{{Name: "tagkey", Value: "tagvalue"}})
	assertServerMatchesExpected(t, server, buf, "sample.thing:4|c|#global,tagkey:tagvalue,host:test_hostname")
}

func assertServerMatchesExpected(t *testing.T, server *net.UDPConn, buf []byte, expected string) {
	n, _ := server.Read(buf)
	msg := buf[:n]
	if string(msg) != expected {
		t.Fatalf("Line %s does not match expected: %s", string(msg), expected)
	}
}
