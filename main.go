package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/equelin/gounity"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	//Create prometheus registry
	reg := prometheus.NewPedanticRegistry()
	log.Print("main.go:main - Loading Config and Metrics")

	exporter, unityClients := readConfig("./config.json")

	//Check if pool metrics are required
	poolMetrics := make([]*prometheus.GaugeVec, 0)
	if exporter.Pools {
		labels := []string{"unity", "id", "name"}
		poolFields := []string{"sizefree", "sizeTotal", "sizeUsed", "sizeSubscribed"}
		for _, field := range poolFields {
			poolMetric := prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					//Namespace: "our_company",
					//Subsystem: "blob_storage",
					Name: "pool_" + field + "_" + "bytes",
					Help: "Bytes of pool " + field,
				},
				labels,
			)
			poolMetrics = append(poolMetrics, poolMetric)
			prometheus.WrapRegistererWith(prometheus.Labels{}, reg).MustRegister(poolMetric)
		}
	}

	storageMetrics := make([]*prometheus.GaugeVec, 0)
	if exporter.StorageResources {
		labels := []string{"unity", "storageresource", "storageresourcename"}
		storageResourceFields := []string{"sizeAllocated", "sizeTotal", "sizeUsed"}
		for _, field := range storageResourceFields {

			storageMetric := prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					//Namespace: "our_company",
					//Subsystem: "blob_storage",
					Name: "storageresource_" + field + "_bytes",
					Help: "Bytes of Storageresource " + field,
				},
				labels,
			)
			storageMetrics = append(storageMetrics, storageMetric)
			prometheus.WrapRegistererWith(prometheus.Labels{}, reg).MustRegister(storageMetric)
		}
	}

	//Prepare additional metrics
	metrics := readMetrics("./unity_metrics.json")

	//Get selected Metrics from metrics json
	selectedMetrics := make([]Metric, 0)
	for _, metric := range metrics {
		for _, path := range exporter.Metrics {
			if metric.PromPath == path {
				//Add Prometheus.Desc to metric
				metric.addPrometheusDesc()
				metric.addPrometheusGaugeVec(reg)
				prometheus.WrapRegistererWith(prometheus.Labels{}, reg).MustRegister(metric.PromGauge)
				selectedMetrics = append(selectedMetrics, metric)
			}
		}
	}

	unityCollectors := make([]UnityCollector, 0)
	//Create Unity Session
	for _, u := range unityClients {
		log.Print("main.go:main - Create unity Session: " + u.IP)
		session, err := gounity.NewSession(u.IP, true, u.User, u.Password)
		if err != nil {
			log.Fatal(err)
		}
		u.Session = *session
		//defer session.CloseSession()

		// Get system informations
		System, err := session.GetbasicSystemInfo()
		if err != nil {
			log.Fatal(err)
		} else {
			// Store the name of the Unity
			u.Name = System.Entries[0].Content.Name
		}
		session.CloseSession()
		log.Print("main.go:main - Unity Name: " + u.Name)
		unityCollectors = append(unityCollectors, NewUnityCollector(u, selectedMetrics, exporter, poolMetrics, storageMetrics))
	}

	log.Print(len(unityCollectors))

	go func() {
		for {
			for _, uc := range unityCollectors {
				session, err := gounity.NewSession(uc.Unity.IP, true, uc.Unity.User, uc.Unity.Password)
				if err != nil {
					log.Fatal(err)
				}
				uc.Unity.Session = *session
				//log.Print(uc.Unity.Name)
				if exporter.Pools {
					uc.CollectPoolMetrics()
				}
				if exporter.StorageResources {
					uc.CollectStorageResourceMetrics()
				}
				uc.CollectMetrics()
				uc.Unity.Session.CloseSession()
			}
			time.Sleep(time.Duration(exporter.Interval) * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	log.Fatal(http.ListenAndServe(strconv.Itoa(exporter.Port), nil))
}
