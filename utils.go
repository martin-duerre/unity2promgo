package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func readConfig(configPath string) (Exporter, []Unity) {
	// Open our jsonFile
	jsonFile, err := os.Open(configPath)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	log.Print("Successfully opened", configPath)
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var config Config
	json.Unmarshal([]byte(byteValue), &config)

	return config.Exporter, config.UnityClients
}

func readMetrics(metricsPath string) []Metric {
	// Open our jsonFile
	jsonFile, err := os.Open(metricsPath)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	log.Print("Successfully opened", metricsPath)
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var metrics Metrics
	json.Unmarshal([]byte(byteValue), &metrics)

	return metrics.Metrics
}

//Config is an array containing RestAPI client information for all monitored Unity Arrays
type Config struct {
	Exporter     Exporter `json:"exporter"`
	UnityClients []Unity  `json:"unitys"`
}

//Exporter represents a single Unity RestAPI client
type Exporter struct {
	Port             int      `json:"port"`
	Interval         int      `json:"interval"`
	Metrics          []string `json:"metrics"`
	Pools            bool     `json:"pools"`
	StorageResources bool     `json:"storage_resources"`
}
