package main

import (
	"strings"

	"gopkg.in/yaml.v2"
)

// DimensionFilter is a map of dimension names
type DimensionFilter map[string]bool

// MetricFilter is a filter specification for a set of metrics
type MetricFilter struct {
	DimensionFilters DimensionFilter `yaml:"dimensions"`
}

// RulesFilter is a collections of filters and configurations
type RulesFilter struct {
	MaxDimensions               int                      `yaml:"maxDimensions"`
	GlobalMetricAllowDefault    bool                     `yaml:"defaultMetricAllow"`
	GlobalDimensionAllowDefault bool                     `yaml:"defaultDimensionAllow"`
	MetricsFilter               map[string]*MetricFilter `yaml:"metrics"`
	StartsWithFilters           []string                 `yaml:"startsWithFilters"`
	GlobalDimensionsFilter      DimensionFilter          `yaml:"dimensions"`
}

var (
	rulesFilter *RulesFilter
)

func isMetricAllowed(metricName string) bool {
	_, exists := rulesFilter.MetricsFilter[metricName]

	if exists {
		return !rulesFilter.GlobalMetricAllowDefault
	}

	for _, startsWithFilter := range rulesFilter.StartsWithFilters {
		if strings.HasPrefix(metricName, startsWithFilter) {
			return true
		}
	}

	return rulesFilter.GlobalMetricAllowDefault
}

func isDimensionAllowed(metricName string, metricDimension string) bool {
	var exists bool

	// if a metric filter specifies a filter on this dimension, that takes precedence
	var filter *MetricFilter
	filter, exists = rulesFilter.MetricsFilter[metricName]
	// if metric name exists, let's dive in deeper
	if exists && filter != nil {
		// if dimension filtering criteria exists, let's dive in deeper
		filterValue, exists := filter.DimensionFilters[metricDimension]
		if exists {
			return filterValue
		}
	}

	// if global dimension filter exists, allow it to deliver its opinion
	var allow bool
	allow, exists = rulesFilter.GlobalDimensionsFilter[metricDimension]
	if exists {
		return allow
	}

	// if no rules have applied thus far, let 'er through!
	return rulesFilter.GlobalDimensionAllowDefault
}

// LoadRules is a helper function for loading filter rules from YAML
func LoadRules(rulesYaml []byte) error {
	rulesFilter = nil
	err := yaml.Unmarshal(rulesYaml, &rulesFilter)

	if rulesFilter == nil {
		rulesFilter = &RulesFilter{}
	}

	if rulesFilter.MaxDimensions == 0 {
		rulesFilter.MaxDimensions = 50
	}

	if err == nil {
		logger.TraceInfof("Using ruleset: %+v", rulesFilter)
	}

	return err
}
