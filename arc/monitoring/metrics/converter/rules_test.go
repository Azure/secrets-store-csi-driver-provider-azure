package converter

import (
	"strings"
	"testing"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/arc/monitoring/metrics/log"

	"github.com/MakeNowJust/heredoc"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
)

func init() {
	logger = log.GetContextlessTraceLogger()
}

func Test_PrefixSupport(t *testing.T) {
	rulesYaml := heredoc.Doc(`
			defaultMetricAllow: false
			defaultDimensionAllow: true
			startsWithFilters: 
				- 'overlays_'
				- 'something_'
			metrics:
				kube_pod_info:
		`)

	rulesYaml = strings.Replace(rulesYaml, "\t", " ", -1)

	err := LoadRules([]byte(rulesYaml))
	assert.Nil(t, err)

	assert.Equal(t, 2, len(rulesFilter.StartsWithFilters))

	assert.Equal(t, true, isMetricAllowed("overlays_yes"))
	assert.Equal(t, true, isMetricAllowed("overlays_"))
	assert.Equal(t, false, isMetricAllowed(""))
	assert.Equal(t, false, isMetricAllowed("over"))
	assert.Equal(t, false, isMetricAllowed("no_overlays_no"))
	assert.Equal(t, true, isMetricAllowed("something_fun"))
}

var _ = Describe("PromMdmConverter Suite", func() {
	It("should allow all metrics by default, pick and choose certain metrics to drop", func() {
		// dimension configurability is mutually exclusive from this logic
		rulesFilter = &RulesFilter{
			GlobalMetricAllowDefault: true,
			MetricsFilter: map[string]*MetricFilter{
				"kube_pod_info":              {},
				"kube_node_info":             {},
				"kube_node_status_condition": {},
			},
		}

		// disallow specified metric
		Expect(isMetricAllowed("kube_node_info")).To(BeFalse())

		// allow unspecified metric
		Expect(isMetricAllowed("kube_pod_status_ready")).To(BeTrue())
	})

	It("should not allow any metrics by default, pick and choose metrics to send", func() {
		// dimension configurability is mutually exclusive from this logic
		rulesFilter = &RulesFilter{
			GlobalMetricAllowDefault: false,
			MetricsFilter: map[string]*MetricFilter{
				"kube_pod_info":              {},
				"kube_node_info":             {},
				"kube_node_status_condition": {},
			},
		}

		// allow specified metric
		Expect(isMetricAllowed("kube_node_info")).To(BeTrue())

		// disallow unspecified metric
		Expect(isMetricAllowed("kube_pod_status_ready")).To(BeFalse())
	})

	It("should allow all dimensions by default, pick and choose certain dimensions to drop", func() {
		// metric configurability is mutually exclusive from this logic
		rulesFilter = &RulesFilter{
			GlobalDimensionAllowDefault: true,
			GlobalDimensionsFilter: map[string]bool{
				"job": false,
			},
			MetricsFilter: map[string]*MetricFilter{
				"kube_pod_info":  {DimensionFilters: map[string]bool{"chart": false}},
				"kube_node_info": {DimensionFilters: map[string]bool{"job": true}},
			},
		}

		// dimension that has been globally denylisted shouldn't show up
		Expect(isDimensionAllowed("kube_pod_info", "job")).To(BeFalse())

		// dimension that has been locally denylisted shouldn't show up
		Expect(isDimensionAllowed("kube_pod_info", "chart")).To(BeFalse())

		// dimension that has been globally denylisted but locally allowlisted should show up
		Expect(isDimensionAllowed("kube_node_info", "job")).To(BeTrue())

		// dimension that hasn't been globally listed should show up
		Expect(isDimensionAllowed("kube_node_info", "chart")).To(BeTrue())
	})

	It("should not allow any dimensions by default, pick and choose certain dimensions to send", func() {
		// metric configurability is mutually exclusive from this logic
		rulesFilter = &RulesFilter{
			GlobalDimensionAllowDefault: false,
			GlobalDimensionsFilter: map[string]bool{
				"job": true,
			},
			MetricsFilter: map[string]*MetricFilter{
				"kube_pod_info":  {DimensionFilters: map[string]bool{"chart": true}},
				"kube_node_info": {DimensionFilters: map[string]bool{"job": false}},
			},
		}

		Expect(isDimensionAllowed("kube_pod_info", "job")).To(BeTrue())
		Expect(isDimensionAllowed("kube_pod_info", "chart")).To(BeTrue())

		Expect(isDimensionAllowed("kube_node_info", "job")).To(BeFalse())
	})

	It("should apply rules from an allowlist metric + denylist dimension rules file", func() {
		// dimension "job" should not show up in any metric except for kube_node_info
		// dimension "chart" should not show up in kube_pod_info
		// metric kube_pod_info should be sent
		// metric kube_node_status_condition should not be sent

		rulesYaml := `
			defaultMetricAllow: false
			defaultDimensionAllow: true
			dimensions:
				job: false
			metrics:
				kube_pod_info:
					dimensions:
						chart: false
				kube_node_info:
					dimensions:
						job: true
				kube_pod_status_phase:
		`

		rulesYaml = strings.Replace(rulesYaml, "\t", " ", -1)

		err := LoadRules([]byte(rulesYaml))

		Expect(err).To(BeNil())
		Expect(rulesFilter.GlobalMetricAllowDefault).To(BeFalse())
		Expect(rulesFilter.GlobalDimensionAllowDefault).To(BeTrue())

		Expect(isMetricAllowed("kube_pod_info")).To(BeTrue())
		Expect(isDimensionAllowed("kube_pod_info", "chart")).To(BeFalse())
		Expect(isDimensionAllowed("kube_pod_info", "job")).To(BeFalse())

		Expect(isMetricAllowed("kube_node_info")).To(BeTrue())
		Expect(isDimensionAllowed("kube_node_info", "job")).To(BeTrue())
		Expect(isDimensionAllowed("kube_node_info", "chart")).To(BeTrue())

		Expect(isMetricAllowed("kube_pod_status_phase")).To(BeTrue())
		Expect(isDimensionAllowed("kube_pod_status_phase", "job")).To(BeFalse())

		Expect(isMetricAllowed("kube_node_status_condition")).To(BeFalse())
	})

	It("should apply rules from a allowlist metric + allowlist dimension rules file", func() {
		// dimension "job" should show up in all metrics except for kube_node_info (shouldn't show up)
		// dimension "chart" should show up in only kube_pod_info
		// metric kube_pod_info should be sent
		// metric kube_node_status_condition should not be sent

		rulesYaml := heredoc.Doc(`
			defaultMetricAllow: false
			defaultDimensionAllow: false
			dimensions:
				job: true
			metrics:
				kube_pod_info:
					dimensions:
						chart: true
				kube_node_info:
					dimensions:
						job: false
				kube_pod_status_phase:
		`)

		rulesYaml = strings.Replace(rulesYaml, "\t", " ", -1)

		err := LoadRules([]byte(rulesYaml))

		Expect(err).To(BeNil())
		Expect(rulesFilter.GlobalMetricAllowDefault).To(BeFalse())
		Expect(rulesFilter.GlobalDimensionAllowDefault).To(BeFalse())

		Expect(isMetricAllowed("kube_pod_info")).To(BeTrue())
		Expect(isDimensionAllowed("kube_pod_info", "chart")).To(BeTrue())
		Expect(isDimensionAllowed("kube_pod_info", "job")).To(BeTrue())

		Expect(isMetricAllowed("kube_node_info")).To(BeTrue())
		Expect(isDimensionAllowed("kube_node_info", "job")).To(BeFalse())
		Expect(isDimensionAllowed("kube_node_info", "chart")).To(BeFalse())

		Expect(isMetricAllowed("kube_pod_status_phase")).To(BeTrue())
		Expect(isDimensionAllowed("kube_pod_status_phase", "job")).To(BeTrue())

		Expect(isMetricAllowed("kube_node_status_condition")).To(BeFalse())
	})

	It("should apply rules from a allowlist metric + allowlist dimension rules file", func() {
		// metric kube_pod_info should not be sent
		// metric kube_node_info should not be sent
		// metric kube_pod_status_phase should not be sent
		// metric kube_node_status_condition should be sent

		rulesYaml := heredoc.Doc(`
			defaultMetricAllow: true
			defaultDimensionAllow: true
			dimensions:
				job: false
			metrics:
				kube_pod_info:
					dimensions:
						chart: false
				kube_node_info:
					dimensions:
						job: true
				kube_pod_status_phase:
		`)

		rulesYaml = strings.Replace(rulesYaml, "\t", " ", -1)

		err := LoadRules([]byte(rulesYaml))

		Expect(err).To(BeNil())
		Expect(rulesFilter.GlobalMetricAllowDefault).To(BeTrue())
		Expect(rulesFilter.GlobalDimensionAllowDefault).To(BeTrue())

		Expect(isMetricAllowed("kube_pod_info")).To(BeFalse())
		// dimensions for disallowed metrics don't make much sense but we'll test them for correctness
		Expect(isDimensionAllowed("kube_pod_info", "chart")).To(BeFalse())
		Expect(isDimensionAllowed("kube_pod_info", "job")).To(BeFalse())

		Expect(isMetricAllowed("kube_node_info")).To(BeFalse())
		// dimensions for disallowed metrics don't make much sense but we'll test them for correctness
		Expect(isDimensionAllowed("kube_node_info", "job")).To(BeTrue())
		Expect(isDimensionAllowed("kube_node_info", "chart")).To(BeTrue())

		Expect(isMetricAllowed("kube_pod_status_phase")).To(BeFalse())

		Expect(isMetricAllowed("kube_node_status_condition")).To(BeTrue())
	})

	It("should apply rules from a denylist metric + allowlist dimension rules file", func() {
		// dimension "job" should show up in all metrics
		// dimension "chart" should only show up in kube_pod_info
		// metric kube_pod_info should not be sent
		// metric kube_fake_something_else should be sent

		rulesYaml := heredoc.Doc(`
			defaultMetricAllow: true
			defaultDimensionAllow: false
			dimensions:
				job: true
			metrics:
				kube_pod_info:
					dimensions:
						chart: true
				kube_node_info:
					dimensions:
						job: false
				kube_pod_status_phase:
		`)

		rulesYaml = strings.Replace(rulesYaml, "\t", " ", -1)

		err := LoadRules([]byte(rulesYaml))

		Expect(err).To(BeNil())
		Expect(rulesFilter.GlobalMetricAllowDefault).To(BeTrue())
		Expect(rulesFilter.GlobalDimensionAllowDefault).To(BeFalse())

		Expect(isMetricAllowed("kube_pod_info")).To(BeFalse())
		// dimensions for disallowed metrics don't make much sense but we'll test them for correctness
		Expect(isDimensionAllowed("kube_pod_info", "chart")).To(BeTrue())
		Expect(isDimensionAllowed("kube_pod_info", "job")).To(BeTrue())

		Expect(isMetricAllowed("kube_node_info")).To(BeFalse())
		// dimensions for disallowed metrics don't make much sense but we'll test them for correctness
		Expect(isDimensionAllowed("kube_node_info", "job")).To(BeFalse())
		Expect(isDimensionAllowed("kube_node_info", "chart")).To(BeFalse())

		Expect(isMetricAllowed("kube_pod_status_phase")).To(BeFalse())

		Expect(isMetricAllowed("kube_node_status_condition")).To(BeTrue())
	})

	It("should be able to apply an 'incomplete' rules file", func() {
		rulesYaml := heredoc.Doc(`
			defaultMetricAllow: true
			defaultDimensionAllow: true
			dimensions:
				chart: false
				job: false
		`)

		rulesYaml = strings.Replace(rulesYaml, "\t", " ", -1)

		err := LoadRules([]byte(rulesYaml))

		Expect(err).To(BeNil())
		Expect(rulesFilter.GlobalMetricAllowDefault).To(BeTrue())
		Expect(rulesFilter.GlobalDimensionAllowDefault).To(BeTrue())

		Expect(rulesFilter.GlobalDimensionsFilter["chart"]).To(BeFalse())
		Expect(rulesFilter.GlobalDimensionsFilter["job"]).To(BeFalse())

		Expect(isMetricAllowed("kube_pod_info")).To(BeTrue())
		Expect(isDimensionAllowed("kube_pod_info", "chart")).To(BeFalse())
		Expect(isDimensionAllowed("kube_pod_info", "job")).To(BeFalse())

		Expect(isDimensionAllowed("kube_pod_info", "instance")).To(BeTrue())

		Expect(rulesFilter.MaxDimensions).To(Equal(50))
	})

	It("should be able to load an empty rules file", func() {
		rulesYaml := heredoc.Doc(`
		`)

		rulesYaml = strings.Replace(rulesYaml, "\t", " ", -1)

		err := LoadRules([]byte(rulesYaml))
		Expect(err).To(BeNil())
		Expect(rulesFilter.MaxDimensions).To(Equal(50))
	})

	It("should allowlist some metrics", func() {
		rulesYaml := heredoc.Doc(`
			defaultMetricAllow: false
			defaultDimensionAllow: true
			metrics:
				kube_pod_info:
				kube_node_info:
				kube_node_status_condition:
				kube_pod_status_ready:
		`)

		rulesYaml = strings.Replace(rulesYaml, "\t", " ", -1)

		err := LoadRules([]byte(rulesYaml))

		Expect(err).To(BeNil())
		Expect(rulesFilter.GlobalMetricAllowDefault).To(BeFalse())
		Expect(rulesFilter.GlobalDimensionAllowDefault).To(BeTrue())

		Expect(isMetricAllowed("kube_pod_info")).To(BeTrue())
		Expect(isMetricAllowed("kube_node_info")).To(BeTrue())
		Expect(isMetricAllowed("kube_node_status_condition")).To(BeTrue())
		Expect(isMetricAllowed("kube_pod_status_ready")).To(BeTrue())

		Expect(isMetricAllowed("kube_pod_status_phase")).To(BeFalse())

		Expect(filterMetadata(&metricMetadata{Metric: "kube_pod_info"})).To(BeTrue())

		Expect(rulesFilter.MaxDimensions).To(Equal(50))
	})
})
