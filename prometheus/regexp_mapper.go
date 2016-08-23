package prometheus

import (
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type RegexpMappingRule struct {
	Pattern           string
	NameReplacement   string
	LabelReplacements map[string]string
}

type RegexpMapper struct {
	RegexpMappingRules []RegexpMappingRule
	regexps            map[string]*regexp.Regexp
}

func NewRegexpMapper(rules []RegexpMappingRule) *RegexpMapper {
	m := &RegexpMapper{RegexpMappingRules: rules}
	m.regexps = make(map[string]*regexp.Regexp)
	for _, rule := range rules {
		m.regexps[rule.Pattern] = regexp.MustCompile(rule.Pattern)
	}
	return m
}

func (m *RegexpMapper) MapMetric(parts []string) (string, prometheus.Labels, bool) {

	joined := strings.Join(parts, "_")

	for _, rule := range m.RegexpMappingRules {
		if matches := m.regexps[rule.Pattern].FindStringSubmatchIndex(joined); matches != nil {
			name := string(m.regexps[rule.Pattern].ExpandString(nil, rule.NameReplacement, joined, matches))
			labels := make(map[string]string)
			for label, labelReplacement := range rule.LabelReplacements {
				labelReplaced := string(m.regexps[rule.Pattern].ExpandString(nil, label, joined, matches))
				labels[labelReplaced] = string(m.regexps[rule.Pattern].ExpandString(nil, labelReplacement, joined, matches))
			}
			return name, labels, true
		}
	}

	return "", nil, false
}
