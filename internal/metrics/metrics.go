// Package metrics is used to register and expose metrics for the application.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const defaultNamespace = "mcp_gateway"

var (
	ToolsCalledGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: defaultNamespace + "_tools_called",
			Help: "Current tools called by name",
		},
		[]string{"tool"},
	)

	ToolsCallErrorsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: defaultNamespace + "_tools_call_errors",
			Help: "Current tools call errors by name",
		},
		[]string{"tool"},
	)

	ToolsCallSuccessGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: defaultNamespace + "_tools_call_success",
			Help: "Current tools call success by name",
		},
		[]string{"tool"},
	)

	CustomGaugeVecMetrics = []*prometheus.GaugeVec{
		ToolsCalledGauge,
		ToolsCallErrorsGauge,
		ToolsCallSuccessGauge,
	}

	CustomCounterMetrics = []prometheus.Counter{}

	CustomGaugeMetrics = []prometheus.Collector{}
)

type Metrics struct {
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) RegisterCustomMetrics() error {
	for _, metric := range CustomGaugeVecMetrics {
		if err := prometheus.DefaultRegisterer.Register(metric); err != nil {
			return err
		}
	}

	for _, metric := range CustomCounterMetrics {
		if err := prometheus.DefaultRegisterer.Register(metric); err != nil {
			return err
		}
	}

	for _, metric := range CustomGaugeMetrics {
		if err := prometheus.DefaultRegisterer.Register(metric); err != nil {
			return err
		}
	}

	return nil
}
