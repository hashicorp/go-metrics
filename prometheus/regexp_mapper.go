package prometheus

import (
	"fmt"
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

func replace(input string, matches []string) string {
	replacementPairs := make([]string, len(matches)*2)
	i := 0
	for j, match := range matches[1:] {
		replacementPairs[i] = fmt.Sprintf("$%d", j+1)
		replacementPairs[i+1] = match
		i += 2
	}
	return strings.NewReplacer(replacementPairs...).Replace(input)
}

func (m *RegexpMapper) MapMetric(parts []string) (string, prometheus.Labels, bool) {

	joined := strings.Join(parts, "_")

	for _, rule := range m.RegexpMappingRules {
		if matches := m.regexps[rule.Pattern].FindStringSubmatch(joined); matches != nil {
			name := replace(rule.NameReplacement, matches)
			labels := make(map[string]string)
			for label, labelReplacement := range rule.LabelReplacements {
				labelReplaced := replace(label, matches)
				labels[labelReplaced] = replace(labelReplacement, matches)
			}
			return name, labels, true
		}
	}

	return "", nil, false
}
