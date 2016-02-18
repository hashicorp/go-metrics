package metrics

import (
	"time"
	"sync"
	"io/ioutil"
	"fmt"
	"bytes"
	log "github.com/Sirupsen/logrus"
	"sort"
	"errors"
)

type AggregateSampleParam struct {
	type_     AggregateSampleParamType
	fmtString string
}
type AggregateSampleParamType int

const (
	Count AggregateSampleParamType = iota
	Throughput
	Mean
	Min
	Max
	Sum
	SumSq
	Stddev
	LastUpdated
)

func CreateASParam(type_ AggregateSampleParamType) AggregateSampleParam {
	return CreateASParamWithFmt(type_, "")
}

func CreateASParamWithFmt(type_ AggregateSampleParamType, fmtString string) AggregateSampleParam {
	res := AggregateSampleParam{type_, fmtString}
	return res
}

var asParamNames []string = []string{"count", "throughput", "mean", "min", "max", "sum", "sumsq", "stddev", "lastupdt"}
var asParamFmtStrings []string = []string{"%d", "%0.3f", "%0.3f", "%0.3f", "%0.3f", "%0.3f", "%0.3f", "%0.3f", "%s"}

func (this *AggregateSampleParam) Name() string {
	return asParamNames[this.type_]
}

func (this *AggregateSampleParam) defaultFmtString() string {
	return asParamFmtStrings[this.type_]
}

func (this *AggregateSampleParam) value(a *AggregateSample) interface{} {
	// TODO: Maybe implement: func (this *AggregateSample) Value(type_ AggregateSampleParamType) interface{}
	switch this.type_ {
	case Count:
		return a.Count
	case Throughput:
		return a.Throughput
	case Mean:
		return a.Mean()
	case Min:
		return a.Min
	case Max:
		return a.Max
	case Sum:
		return a.Sum
	case SumSq:
		return a.SumSq
	case Stddev:
		return a.Stddev()
	case LastUpdated:
		return a.LastUpdated
	default:
		panic(errors.New(fmt.Sprintf("Unknown value %v", this.type_)))
	}
}

func (this *AggregateSampleParam) FormatValue(a *AggregateSample) string {
	fmtString := this.fmtString
	if fmtString == "" {
		fmtString = this.defaultFmtString()
	}
	v := this.value(a)
	return fmt.Sprintf(fmtString, v)
}

type AggregateSampleFormatter struct {
	params []AggregateSampleParam
}

var defaultFormatter = AggregateSampleFormatter{
	[]AggregateSampleParam{ AggregateSampleParam{Count,""}, AggregateSampleParam{Mean,""}, AggregateSampleParam{Min,""}, AggregateSampleParam{Max,""} }}

type InmemFileDumper struct {
	ticker         *time.Ticker
	inm            *InmemSink
	outputFile     string

	stop           bool
	stopCh         chan struct{}
	stopLock       sync.Mutex

	userFormatters map[string] *AggregateSampleFormatter
}


func NewInmemFileDumper(inmem *InmemSink, dumpInterval time.Duration, outputFile string) *InmemFileDumper {
	i := &InmemFileDumper{
		ticker: time.NewTicker(dumpInterval),
		inm:    inmem,
		outputFile:      outputFile,
		stopCh: make(chan struct{}),
		userFormatters: make(map[string] *AggregateSampleFormatter),
	}
	go i.run()
	return i
}

func (i *InmemFileDumper) SetAggregateSampleFormatter(key string, paramsTypesToShow ...AggregateSampleParamType) {
	var sps []AggregateSampleParam = make([]AggregateSampleParam, len(paramsTypesToShow), len(paramsTypesToShow))
	for i := 0; i < len(paramsTypesToShow); i++ {
		type_ := paramsTypesToShow[i]
		sps[i] = CreateASParam(type_)
	}
	i.userFormatters[key] = &AggregateSampleFormatter{sps}
}

func (i *InmemFileDumper) SetAggregateSampleFormatter2(key string, paramsToShow ...AggregateSampleParam) {
	i.userFormatters[key] = &AggregateSampleFormatter{paramsToShow}
}

func (i *InmemFileDumper) Stop() {
	i.stopLock.Lock()
	defer i.stopLock.Unlock()

	if i.stop {
		return
	}
	i.stop = true
	close(i.stopCh)
	i.ticker.Stop()
}

func (i *InmemFileDumper) run() {
	for {
		select {
		case <-i.ticker.C:
			i.dumpStats()
		case <-i.stopCh:
			return
		}
	}
}

func (i *InmemFileDumper) dumpStats() {
	buf := bytes.NewBuffer(nil)
	data := i.inm.Data()

	// Get the prev period. Because the last period is still being aggregated
	if len(data) >= 1 {
		intv := data[len(data) - 1]

		intv.RLock()
		for name, val := range intv.Gauges {
			fmt.Fprintf(buf, "%v = %v\n", name, val)
		}

		for name, vals := range intv.Points {
			for _, val := range vals {
				fmt.Fprintf(buf, "%v = %v\n", name, val)
			}
		}

		names := sortedKeys(intv.Counters)
		for n := range names {
			name := names[n]
			format := i.getFormatter(name)
			agg := intv.Counters[name]
			dumpAggregateSampleToBuf(buf, name, format, agg)
		}

		names = sortedKeys(intv.Samples)
		for n := range names {
			name := names[n]
			format := i.getFormatter(name)
			agg := intv.Samples[name]
			dumpAggregateSampleToBuf(buf, name, format, agg)
		}
		intv.RUnlock()
	}

	err := ioutil.WriteFile(i.outputFile, buf.Bytes(), 0666)
	if err != nil {
		log.Errorf("Cannot dumpStats to file %s, reason %v", i.outputFile, err)
	}
}

func (i *InmemFileDumper) getFormatter(name string) *AggregateSampleFormatter {
	format, ok := i.userFormatters[name]
	if !ok {
		format = &defaultFormatter
	}
	return format
}

func sortedKeys(m map[string]*AggregateSample) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func dumpAggregateSampleToBuf(buf *bytes.Buffer, name string, formatter *AggregateSampleFormatter, a *AggregateSample) {
	for i := 0; i < len(formatter.params); i++ {
		subParam := formatter.params[i]
		fmt.Fprintf(buf, "%v.%v = %s\n", name, subParam.Name(), subParam.FormatValue(a))
	}
}

func TimeNowUnixMillis() int64 {
	return time.Now().UnixNano() / 1000000
}