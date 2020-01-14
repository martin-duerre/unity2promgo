# unity2promgo
Prometheus Exporter for DellEMC Unity

Unity2Prom is a tool written in Go for exporting usage and performance metrics from Dell EMC Unity storage arrays to prometheus.

For RestAPI connectivity to the different Unity arrays [gounity by Equelin](https://github.com/equelin/gounity.git) is used.

## Configuration
### Configure the Exporter
Before starting the exporter it is required to update *config.json* with values specific to your environment:
```json
{
  "exporter": {
    "port": ":8080",
    "interval": 60,
    "pools": true,
    "storage_resources": true,
    "metrics": ["sp_cifs_global_basic_writeBytesRate", "sp_cifs_smb1_usage_currentConnections","sp_net_device_pktsInRate","sp_net_device_pktsOut"]
  },
  "unitys": [
    {
      "ip": "192.168.2.190",
      "port": "",
      "user": "monitor",
      "password": "Monitor123!"
    }
  ]
}
```
*Unitys** is a list of all arrays for which data will be collected.
For the user it is sufficient to be a *Operator* User with read-only access.

**Exporter** conifgures the exporter and specifies the data that will be collected.
**!!!The interval in which metrics are collected should match the Prometheus scraping interval!!!**
Metrics can be added by adding the metric's *prom_path* to the metrics array.
Possible metrics and their description can be found in *unity_metrics.json*:

```json
  {
    "id": 10170,
    "name": "Read_Hit_Only Miss requests",
    "path": "sp.*.blockCache.flu.*.readHitOnlyMiss",
    "prom_path": "sp_blockCache_flu_readHitOnlyMiss",
    "type": 3,
    "description": "Number of Read_Hit_Only Miss requests",
    "isHistoricalAvailable": false,
    "isRealtimeAvailable": true,
    "unitDisplayString": "requests"
  },
```

### Configure Prometheus Scrape Job

In order to configure Prometheus to scrape the newly added metric it is requried to add another scrape_config entriy to the 
promehteus configuration file:
```yaml
scrape_configs:
- job_name: unity2go  
  scrape_interval: 60s
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: http
  static_configs:
  - targets:
    - <unity.fqdn|ip_adress>:9090    
```

## Installation
After installing go and cloning this repositroy it is requried to install the following go dependencies:
- github.com/equelin/gounity
- github.com/prometheus/client_golang/prometheus
- github.com/prometheus/client_golang/prometheus/promhttp

This can be done using go *get -u <dependency_name>*

*Start the collector*
In the project base directory run
**go build**
**go run main.go utils.go unitycollector.go**

