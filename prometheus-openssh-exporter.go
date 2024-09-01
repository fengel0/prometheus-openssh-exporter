package main

import (
	"log"
	"net/http"
	"os"
	journalreader "prometheus-openssh-exporter/journal_reader"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusInformation struct {
	successFull prometheus.Gauge
	failed      prometheus.Gauge
	all         prometheus.Gauge
	hosts       prometheus.GaugeVec
}

type EnvConfig struct {
	journalFile               string
	timeSpaneToCountInMinutes int
	readIntervalInSeconds     int
	port                      int
}

const HOST_STRING = "Host"
const REQUEST_STRING = "Request"
const SUCCESS_STRING = "Successful"
const FAILED_STRING = "Failed"

func getFullJournalLocation(folderSearch string) string {
	files, err := os.ReadDir(folderSearch)
	if err != nil {
		log.Fatal(err)
	}

	if len(files) > 1 {
		panic("more then one journal id?")
	}

	if len(files) < 1 {
		panic("no journal folder found")
	}

	return folderSearch + "/" + files[0].Name() + "/system.journal"
}

func loadEnv() EnvConfig {
	journalLocation := "/var/log/journal"
	timeSpanInMinutes := 30
	port := 8080

	value, found := os.LookupEnv("journal")
	if found {
		journalLocation = value
	}

	journalLocation = getFullJournalLocation(journalLocation)
	println("listen on folder " + journalLocation)

	value, found = os.LookupEnv("port")
	if found {
		i, err := strconv.Atoi(value)
		if err != nil {
			log.Fatal(err)
		}
		port = i
	}

	config := EnvConfig{
		journalFile:               journalLocation,
		timeSpaneToCountInMinutes: timeSpanInMinutes,
		port:                      port,
		readIntervalInSeconds:     15,
	}
	return config
}

func getPrometheusElements() PrometheusInformation {

	keyValueGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ssh_host_connection",
			Help: "Displays a list of hosts sending request",
		},
		[]string{HOST_STRING, REQUEST_STRING},
	)

	// Create a Counter metric
	sshProcessed := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ssh_connection_in_the_last_30_minuts",
		Help: "the total number of ssh connections in the last 30 minutes",
	})

	sshSucessProcessed := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ssh_successful_connection_in_the_last_30_minuts",
		Help: "the total number of successful ssh connections in the last 30 minutes",
	})

	sshFailedProcessed := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ssh_failed_connection_in_the_last_30_minuts",
		Help: "the total number of failed ssh connections in the last 30 minutes",
	})

	prometheusHistograms := PrometheusInformation{
		successFull: sshSucessProcessed,
		failed:      sshFailedProcessed,
		all:         sshProcessed,
		hosts:       *keyValueGauge,
	}
	return prometheusHistograms
}

func main() {
	config := loadEnv()

	prometheusHistograms := getPrometheusElements()

	// Register the metrics with Prometheus
	prometheus.MustRegister(prometheusHistograms.failed)
	prometheus.MustRegister(prometheusHistograms.successFull)
	prometheus.MustRegister(prometheusHistograms.all)
	prometheus.MustRegister(prometheusHistograms.hosts)

	// start gorutine to check logs parallel
	go logSsh(config.journalFile, config.timeSpaneToCountInMinutes, config.readIntervalInSeconds, &prometheusHistograms)

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":"+strconv.Itoa(config.port), nil)
}

func logSsh(journalLocation string, timeSpanInMinutes int, scrapeIntervalInSeconds int, prometheusInformation *PrometheusInformation) {
	for {
		requests, hosts := journalreader.CountRequest(journalLocation, timeSpanInMinutes)
		prometheusInformation.failed.Set(float64(requests.FailedRequest))
		prometheusInformation.successFull.Set(float64(requests.SuccessFullRequest))
		prometheusInformation.all.Set(float64(requests.SuccessFullRequest + requests.FailedRequest))

		prometheusInformation.hosts.Reset()
		for host, request := range hosts {
			prometheusInformation.hosts.With(prometheus.Labels{HOST_STRING: host, REQUEST_STRING: SUCCESS_STRING}).Set(float64(request.SuccessFullRequest))
			prometheusInformation.hosts.With(prometheus.Labels{HOST_STRING: host, REQUEST_STRING: FAILED_STRING}).Set(float64(request.FailedRequest))
		}

		time.Sleep(time.Duration(scrapeIntervalInSeconds) * time.Second)
	}
}
