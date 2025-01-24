package main

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/noorbala7418/softaculous-traffic-exporter/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var apiKey = os.Getenv("SOFTACULOUS_API_KEY")
var apiPassword = os.Getenv("SOFTACULOUS_API_PASSWORD")
var softaculousUrl = os.Getenv("SOFTACULOUS_URL")
var scrapeTimeEnv = os.Getenv("SOFTACULOUS_SCRAPE_SCHEDULE")
var insecureSSLCheckStatus = os.Getenv("SOFTACULOUS_IGNORE_SSL")
var scrapeTime int
var insecureSSLCheck = false

var (
	totalMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "softaculous_vm_traffic_total",
		Help: "Total traffic",
	}, []string{
		// belongs to which vm
		"server_id", "server_name",
	})

	freeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "softaculous_vm_traffic_free",
		Help: "Free traffic",
	}, []string{
		// belongs to which vm
		"server_id", "server_name",
	})

	usedMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "softaculous_vm_traffic_used",
		Help: "Used traffic",
	}, []string{
		// belongs to which vm
		"server_id", "server_name",
	})

	freePercentMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "softaculous_vm_traffic_free_percent",
		Help: "Free traffic percent",
	}, []string{
		// belongs to which vm
		"server_id", "server_name",
	})

	usedPercentMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "softaculous_vm_traffic_used_percent",
		Help: "Used traffic percent",
	}, []string{
		// belongs to which vm
		"server_id", "server_name",
	})
)

func main() {
	checkEnvs()
	logrus.Info("start softaculous teraffic exporter")
	logrus.Info("scrape every ", scrapeTime, " minutes.")
	c := gocron.NewScheduler(time.UTC)
	c.Every(scrapeTime).Minutes().Do(checkStatus)
	c.StartAsync()

	logrus.Info("function main: cron started.")

	// expose /metrics for prometheus to gather
	http.Handle("/metrics", promhttp.Handler())

	// http server
	logrus.Info("Starting server at port 9153")
	logrus.Info("Server started. Metrics are available at 0.0.0.0:9153/metrics")
	if err := http.ListenAndServe(":9153", nil); err != nil {
		logrus.Fatal("Failed to start metrics server. ", err)
	}
}

// getVmList receives VM ID from json API.
func getVmList() (model.VmList, error) {
	// Start checking site
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSSLCheck},
	}

	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: tr,
	}
	url := softaculousUrl + "?act=listvs&api=json&apikey=" + apiKey + "&apipass=" + apiPassword
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		logrus.Warning("function getVmList: Error in list vms ", err)
		return model.VmList{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)

	if err != nil {
		logrus.Warning("function getVmList: List vms failed. Err: ", err)
		return model.VmList{}, err
	}

	defer resp.Body.Close()
	resBody, _ := io.ReadAll(resp.Body)

	var vmlist model.VmList
	err = json.Unmarshal(resBody, &vmlist)
	if err != nil {
		logrus.Error("function getVmList: Error in unmarshaling vm list. ", err.Error())
		return model.VmList{}, err
	}
	return vmlist, nil
}

//checkStatus extracts traffic from json api and convert it to prometheus metrics.
func checkStatus() {
	vpsID, getVpsListErr := getVmList()
	if getVpsListErr != nil {
		logrus.Error("function checkStatus: Error in getting vps list. ", getVpsListErr)
	}

	logrus.Info("function checkStatus: Server gathered. Account ID is: ", vpsID.Username)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSSLCheck},
	}
	// Start checking site
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: tr,
	}
	url := softaculousUrl + "?act=vpsmanage&api=json&apikey=" + apiKey + "&apipass=" + apiPassword + "&svs=" + vpsID.VMId
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		logrus.Warning("function checkStatus: Error in get vm "+vpsID.VMId+" info ", err)
		totalMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		freeMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		freePercentMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		usedMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		usedPercentMetric.WithLabelValues(vpsID.VMId, "").Set(0)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)

	if err != nil {
		logrus.Warning("function checkStatus: Error in send request ", err)
		totalMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		freeMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		freePercentMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		usedMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		usedPercentMetric.WithLabelValues(vpsID.VMId, "").Set(0)
	}

	defer resp.Body.Close()
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Warning("function checkStatus: Error in read body ", err)
		totalMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		freeMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		freePercentMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		usedMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		usedPercentMetric.WithLabelValues(vpsID.VMId, "").Set(0)
	}

	var vm model.VM
	err = json.Unmarshal(resBody, &vm)
	if err != nil {
		logrus.Error("function checkStatus: Error in unmarshaling vm data. ", err.Error())
		totalMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		freeMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		freePercentMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		usedMetric.WithLabelValues(vpsID.VMId, "").Set(0)
		usedPercentMetric.WithLabelValues(vpsID.VMId, "").Set(0)
	}

	totalMetric.WithLabelValues(vpsID.VMId, vm.VMInformation.Hostname).Set(float64(vm.VMInformation.Bandwidth.TotalTraffic))
	freeMetric.WithLabelValues(vpsID.VMId, vm.VMInformation.Hostname).Set(vm.VMInformation.Bandwidth.FreeTraffic)
	freePercentMetric.WithLabelValues(vpsID.VMId, vm.VMInformation.Hostname).Set(vm.VMInformation.Bandwidth.FreeTrafficPercent)
	usedMetric.WithLabelValues(vpsID.VMId, vm.VMInformation.Hostname).Set(vm.VMInformation.Bandwidth.UsedTraffic)
	usedPercentMetric.WithLabelValues(vpsID.VMId, vm.VMInformation.Hostname).Set(vm.VMInformation.Bandwidth.UsedTrafficPercent)
}

// checkEnvs checks required environment variables. If any of them are not defined, then it terminates application.
func checkEnvs() {
	if apiKey == "" {
		logrus.Error("function checkEnvs: Env SOFTACULOUS_API_KEY is not defined.")
		os.Exit(1)
	}
	if apiPassword == "" {
		logrus.Error("function checkEnvs: Env SOFTACULOUS_API_PASSWORD is not defined.")
		os.Exit(1)
	}
	if softaculousUrl == "" {
		logrus.Error("function checkEnvs: Env SOFTACULOUS_URL is not defined.")
		os.Exit(1)
	}
	if scrapeTimeEnv == "" {
		logrus.Warning("function checkEnvs: Env SOFTACULOUS_SCRAPE_SCHEDULE is not defined.")
		scrapeTime = 5
	} else {
		scrapeTime, _ = strconv.Atoi(scrapeTimeEnv)
	}
	if insecureSSLCheckStatus == "" {
		logrus.Warning("function checkEnvs: Env SOFTACULOUS_IGNORE_SSL is not defined. Default is FALSE.")
	} else {
		insecureSSLCheck, _ = strconv.ParseBool(insecureSSLCheckStatus)
	}
}
