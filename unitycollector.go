package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/equelin/gounity"
	"github.com/prometheus/client_golang/prometheus"
)

//Metrics is a List of Metrics
type Metrics struct {
	Metrics []Metric `json:"metrics"`
}

//Metric represents a unity prometheus metric
type Metric struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	PromPath    string `json:"prom_path"`
	Description string `json:"description"`
	Historic    bool   `json:"isHistoricalAvailable"`
	Realtime    bool   `json:"isRealtimeAvailable"`
	Unit        string `json:"unitDisplayString"`
	PromDesc    *prometheus.Desc
	PromGauge   *prometheus.GaugeVec
}

func (m *Metric) addPrometheusDesc() *Metric {
	log.Print("Prometheus Desc: " + m.PromPath)

	//The metric name will be the PromPath + the unit if it's not empty
	metricName := m.PromPath
	if m.Unit != "" {
		metricName = metricName + "_" + m.Unit
	}

	//To generate the labels we look at the metric path
	labels := make([]string, 0)
	labels = append(labels, "unity")

	pathSplit := strings.Split(m.Path, ".")
	//If the unity metric path contains "*" - the element before it will be used as a value identifier
	//sp.*.net.device.*.pktsInRate => 0 -> sp; 2->device
	for i, v := range pathSplit {
		if v == "*" || v == "+" {
			labels = append(labels, pathSplit[i-1])
		}
	}
	m.PromDesc = prometheus.NewDesc(
		metricName,
		m.Description,
		labels,
		nil,
	)
	return m
}

func (m *Metric) addPrometheusGaugeVec(reg prometheus.Registerer) *Metric {
	log.Print("Prometheus Desc: " + m.PromPath)

	//The metric name will be the PromPath + the unit if it's not empty
	metricName := m.PromPath
	if m.Unit != "" {
		metricName = metricName + "_" + m.Unit
	}

	//To generate the labels we look at the metric path
	labels := make([]string, 0)
	labels = append(labels, "unity")

	pathSplit := strings.Split(m.Path, ".")
	//If the unity metric path contains "*" - the element before it will be used as a value identifier
	//sp.*.net.device.*.pktsInRate => 0 -> sp; 2->device
	for i, v := range pathSplit {
		if v == "*" || v == "+" {
			labels = append(labels, pathSplit[i-1])
		}
	}

	m.PromGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			//Namespace: "our_company",
			//Subsystem: "blob_storage",
			Name: metricName,
			Help: m.Description,
		},
		labels,
	)
	return m
}

//Unity represents a single Unity RestAPI client
type Unity struct {
	IP       string `json:"ip"`
	User     string `json:"user"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	Session  gounity.Session
	Name     string
}

//UnityCollector is a prometheus.collector wrapper a unity prometheus metric
type UnityCollector struct {
	Unity          Unity
	Metrics        []Metric
	Exporter       Exporter
	PoolMetrics    []*prometheus.GaugeVec
	StorageMetrics []*prometheus.GaugeVec
}

//NewUnityCollector wraps the Unity and Metrics into a collector
func NewUnityCollector(u Unity, ms []Metric, e Exporter, pm []*prometheus.GaugeVec, sm []*prometheus.GaugeVec) UnityCollector {
	log.Print("unityCollector.go:NewUnityCollector - Unity: " + u.Name)
	uc := UnityCollector{Metrics: ms, Unity: u, Exporter: e, PoolMetrics: pm, StorageMetrics: pm}

	return uc
}

func (uc UnityCollector) CollectMetrics() {
	go func() {
		//Slice of all realtime metrics in order to be possible to handle them in one request
		realtimeMetrics := make([]Metric, 0)
		realtimeMetricPaths := make([]string, 0)
		labels := make([]string, 0)
		labels = append(labels, uc.Unity.Name)

		//Iterate over all metrics
		for _, metric := range uc.Metrics {

			if metric.Realtime {
				//log.Print("UnityCollector - Collector: Realtime Metric - " + metric.PromPath)
				realtimeMetrics = append(realtimeMetrics, metric)
				realtimeMetricPaths = append(realtimeMetricPaths, metric.Path)

			}

			if metric.Historic {
				//	log.Print("UnityCollector - Collector: Historic Metric - " + metric.PromPath)
				//MetricValue, err := uc.Session.GetmetricValue(p)
				MetricValue, err := uc.Unity.Session.GetmetricValue(metric.Path)
				if err != nil {
					log.Print("UnityCollector - Collector: Could not get " + metric.PromPath)
				} else {
					//Historic Metric contians multiple result entries with [0] being the latest
					if MetricValue.Entries[0].Content.Values != nil {
						parseResult(MetricValue.Entries[0].Content.Values.(map[string]interface{}), metric.PromGauge, labels)
					}
				}
			}
		}

		if len(realtimeMetricPaths) != 0 {

			//Get and parse Realtime Metrics
			// TODO: Muss async sein

			log.Print("UnityCollector - Collector: Query Realtime Metrics with interval")
			query, err := uc.Unity.Session.NewMetricRealTimeQuery(realtimeMetricPaths, uint32(uc.Exporter.Interval))
			if err != nil {
				log.Fatal(err)
			}
			// Waiting thforat the sampling of the metrics to be done
			time.Sleep(time.Duration(query.Content.Interval) * time.Second)

			// Get the results of the query
			result, err := uc.Unity.Session.GetMetricRealTimeQueryResult(query.Content.ID)
			if err != nil {
				log.Print("Querying real time metric(s)")
			} else {
				// Parse the results
				for i, v := range result.Entries {
					//Real time metric have only one entry that will be returned for every metric
					//parseResult([]string{uc.Unity.Name}, v.Content.Values.(map[string]interface{}))

					parseResult(v.Content.Values.(map[string]interface{}), realtimeMetrics[i].PromGauge, labels)
				}
			}

		}
	}()
}

func (uc UnityCollector) CollectPoolMetrics() {
	go func() {
		//Slice of all realtime metrics in order to be possible to handle them in one request
		Pools, err := uc.Unity.Session.GetPool()
		log.Print(Pools)
		if err != nil {
			log.Fatal(err)
		} else {
			for _, p := range Pools.Entries {
				labels := make([]string, 0)
				labels = append(labels, uc.Unity.Name)
				labels = append(labels, p.Content.ID)
				labels = append(labels, p.Content.Name)
				uc.PoolMetrics[0].WithLabelValues(labels...).Set(float64(p.Content.SizeFree))
				uc.PoolMetrics[1].WithLabelValues(labels...).Set(float64(p.Content.SizeTotal))
				uc.PoolMetrics[2].WithLabelValues(labels...).Set(float64(p.Content.SizeUsed))
				uc.PoolMetrics[3].WithLabelValues(labels...).Set(float64(p.Content.SizeSubscribed))
			}
		}
	}()
}

func (uc UnityCollector) CollectStorageResourceMetrics() {
	go func() {
		//Slice of all realtime metrics in order to be possible to handle them in one request
		StorageResources, err := uc.Unity.Session.GetStorageResource()
		log.Print(StorageResources)
		if err != nil {
			log.Fatal(err)
		} else {
			for _, sr := range StorageResources.Entries {
				labels := make([]string, 0)
				labels = append(labels, uc.Unity.Name)
				labels = append(labels, sr.Content.ID)
				labels = append(labels, sr.Content.Name)
				uc.StorageMetrics[0].WithLabelValues(labels...).Set(float64(sr.Content.SizeAllocated))
				uc.StorageMetrics[1].WithLabelValues(labels...).Set(float64(sr.Content.SizeTotal))
				uc.StorageMetrics[2].WithLabelValues(labels...).Set(float64(sr.Content.SizeUsed))
			}
		}
	}()
}

//Sample valuesMap
//                "values": {
//                    "spa": {
//												"device1": 100
//										}
//								}
//// TODO: Description
func parseResult(valuesMap map[string]interface{}, promGauge *prometheus.GaugeVec, labels []string) {
	//Current level of the values map
	for key, val := range valuesMap {
		labels := append(labels, key)
		//Switch statement to decicde if current element is anohter map
		//If yes -> further recursion
		//If no  -> print values
		//log.Print(val)
		switch vt := val.(type) {

		//First case is an encaspulated value
		case map[string]interface{}:

			//go donw one more level
			parseResult(
				val.(map[string]interface{}),
				promGauge,
				labels)

		//came to none map value
		case string:
			valstr := fmt.Sprintf("%s", vt)
			val64, _ := strconv.ParseFloat(valstr, 64)
			//log.Print(labels, vt," string ",valstr,val64 )
			promGauge.WithLabelValues(labels...).Set(val64)
		case int32, int64:
			log.Print(labels, vt, " int")
		case float64:
			//log.Print(labels, vt," float" )
			val, _ := val.(float64)
			promGauge.WithLabelValues(labels...).Set(val)
		default:
			log.Print(labels, vt, " default")
			val, _ := val.(float64)
			promGauge.WithLabelValues(labels...).Set(val)

		}
	}
}
