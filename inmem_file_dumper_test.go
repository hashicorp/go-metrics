package metrics

import (
	"testing"
	"time"
	"os"
	"io/ioutil"
	"strings"
	"regexp"
)

func TestInmemFileDumper(t *testing.T) {
	inm := NewInmemSink(100*time.Millisecond, 10*time.Second)

	filename := "./metrics.out"
	fd := NewInmemFileDumper(inm, 100*time.Millisecond, filename)
	defer fd.Stop()

	go func() {
		for {
			// Continiuosly add data points
			inm.SetGauge([]string{"SetGauge", "bar"}, 42)
			inm.EmitKey([]string{"EmitKey", "bar"}, 42)
			inm.IncrCounter([]string{"IncrCounter", "bar"}, 20)
			inm.IncrCounter([]string{"IncrCounter", "bar"}, 22)
			inm.AddSample([]string{"AddSample", "bar"}, 20)
			inm.AddSample([]string{"AddSample", "bar"}, 22)
			time.Sleep(10*time.Millisecond)
		}
	}()

	
	time.Sleep(500*time.Millisecond)
	
	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("file %s didn't created, reason %v", filename, err)
	}
	
	d, err := ioutil.ReadAll(f)
	if len(d) == 0 {
		t.Fatalf("Empty file %s", filename)
	}
	
	str := string(d)
	lines := strings.Split(str, "\n")
	
	anyLineMatches(t, `SetGauge.bar = \d+`, lines)
	anyLineMatches(t, `EmitKey.bar = \d+`, lines)
	anyLineMatches(t, `IncrCounter.bar.count = \d+`, lines)
	anyLineMatches(t, `IncrCounter.bar.mean = \d+`, lines)
	anyLineMatches(t, `IncrCounter.bar.min = \d+`, lines)
	anyLineMatches(t, `IncrCounter.bar.max = \d+`, lines)
	anyLineMatches(t, `AddSample.bar.count = \d+`, lines)
	anyLineMatches(t, `AddSample.bar.mean = \d+`, lines)
	anyLineMatches(t, `AddSample.bar.min = \d+`, lines)
	anyLineMatches(t, `AddSample.bar.max = \d+`, lines)
}

func anyLineMatches(t *testing.T, re string, lines []string) {
	found := false
	for _, line := range lines {
		re := regexp.MustCompile(re)
		if re.MatchString(line) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Cannot find line that matches '%s'", re)
	}
}
