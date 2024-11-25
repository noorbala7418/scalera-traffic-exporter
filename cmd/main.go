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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var apiKey = os.Getenv("SCALERA_API_KEY")
var apiPassword = os.Getenv("SCALERA_API_PASSWORD")
var scaleraUrl = os.Getenv("SCALERA_URL")
var scrapeTimeEnv = os.Getenv("SCALERA_SCRAPE_SCHEDULE")
var insecureSSLCheckStatus = os.Getenv("SCALERA_IGNORE_SSL")
var scrapeTime int
var insecureSSLCheck = false

var (
	// vmCountMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
	// 	Name: "scalera_vm_count",
	// 	Help: "Total vms",
	// }, []string{
	// 	// belongs to wich vm
	// 	"username",
	// })

	totalMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scalera_vm_traffic_total",
		Help: "Total traffic",
	}, []string{
		// belongs to wich vm
		"server_id",
	})

	freeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scalera_vm_traffic_free",
		Help: "Free traffic",
	}, []string{
		// belongs to wich vm
		"server_id",
	})

	usedMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scalera_vm_traffic_used",
		Help: "Used traffic",
	}, []string{
		// belongs to wich vm
		"server_id",
	})

	freePercentMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scalera_vm_traffic_free_percent",
		Help: "Free traffic percent",
	}, []string{
		// belongs to wich vm
		"server_id",
	})

	usedPercentMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scalera_vm_traffic_used_percent",
		Help: "Used traffic percent",
	}, []string{
		// belongs to wich vm
		"server_id",
	})
)

type VmList struct {
	Username string `json:"username"`
	VMId     string `json:"vpsid"`
}

type VM struct {
	VpsID         int    `json:"vpsid"`
	VMInformation VMInfo `json:"info"`
}

type VMInfo struct {
	Hostname  string    `json:"hostname"`
	Bandwidth VMTraffic `json:"bandwidth"`
}

type VMTraffic struct {
	TotalTraffic       int     `json:"limit_gb"`
	UsedTraffic        float64 `json:"used_gb"`
	UsedTrafficPercent float64 `json:"percent"`
	FreeTraffic        float64 `json:"free_gb"`
	FreeTrafficPercent float64 `json:"percent_free"`
}

func main() {
	checkEnvs()
	logrus.Info("start scalera teraffic exporter")
	logrus.Info("scrape every ", scrapeTime, " minutes.")
	c := gocron.NewScheduler(time.UTC)
	c.Every(scrapeTime).Minutes().Do(checkStatus)
	c.StartAsync()

	logrus.Info("cron started.")

	// expose /metrics for prometheus to gather
	http.Handle("/metrics", promhttp.Handler())

	// http server
	logrus.Info("Starting server at port 9153")
	logrus.Info("Server started. Metrics are available at 0.0.0.0:9153/metrics")
	if err := http.ListenAndServe(":9153", nil); err != nil {
		logrus.Fatal("Failed to start metrics server. ", err)
	}
}

func getVmList() (VmList, error) {
	// Start checking site
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSSLCheck},
	}

	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: tr,
	}
	url := scaleraUrl + "?act=listvs&api=json&apikey=" + apiKey + "&apipass=" + apiPassword
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		logrus.Warning("Error in list vms ", err)
		return VmList{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)

	if err != nil {
		logrus.Warning("List vms failed. Err: ", err)
		return VmList{}, err
	}

	defer resp.Body.Close()
	resBody, _ := io.ReadAll(resp.Body)

	var vmlist VmList
	err = json.Unmarshal(resBody, &vmlist)
	if err != nil {
		logrus.Error("Error in unmarshaling vm list. ", err.Error())
		return VmList{}, err
	}
	return vmlist, nil
}

func checkStatus() {
	vpsID, getVpsListErr := getVmList()
	if getVpsListErr != nil {
		logrus.Error("Error in getting vps list. ", getVpsListErr)
	}

	logrus.Info("Server gathered. Account ID is: ", vpsID.Username)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSSLCheck},
	}
	// Start checking site
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: tr,
	}
	url := scaleraUrl + "?act=vpsmanage&api=json&apikey=" + apiKey + "&apipass=" + apiPassword + "&svs=" + vpsID.VMId
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		logrus.Warning("Error in get vm info ", err)
		totalMetric.WithLabelValues(vpsID.VMId).Set(0)
		freeMetric.WithLabelValues(vpsID.VMId).Set(0)
		freePercentMetric.WithLabelValues(vpsID.VMId).Set(0)
		usedMetric.WithLabelValues(vpsID.VMId).Set(0)
		usedPercentMetric.WithLabelValues(vpsID.VMId).Set(0)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)

	if err != nil {
		logrus.Warning("Error in send request ", err)
		totalMetric.WithLabelValues(vpsID.VMId).Set(0)
		freeMetric.WithLabelValues(vpsID.VMId).Set(0)
		freePercentMetric.WithLabelValues(vpsID.VMId).Set(0)
		usedMetric.WithLabelValues(vpsID.VMId).Set(0)
		usedPercentMetric.WithLabelValues(vpsID.VMId).Set(0)
	}

	defer resp.Body.Close()
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Warning("Error in read body ", err)
		totalMetric.WithLabelValues(vpsID.VMId).Set(0)
		freeMetric.WithLabelValues(vpsID.VMId).Set(0)
		freePercentMetric.WithLabelValues(vpsID.VMId).Set(0)
		usedMetric.WithLabelValues(vpsID.VMId).Set(0)
		usedPercentMetric.WithLabelValues(vpsID.VMId).Set(0)
	}

	var vm VM
	err = json.Unmarshal(resBody, &vm)
	if err != nil {
		logrus.Error("Error in unmarshaling vm data. ", err.Error())
		totalMetric.WithLabelValues(vpsID.VMId).Set(0)
		freeMetric.WithLabelValues(vpsID.VMId).Set(0)
		freePercentMetric.WithLabelValues(vpsID.VMId).Set(0)
		usedMetric.WithLabelValues(vpsID.VMId).Set(0)
		usedPercentMetric.WithLabelValues(vpsID.VMId).Set(0)
	}

	totalMetric.WithLabelValues(vpsID.VMId).Set(float64(vm.VMInformation.Bandwidth.TotalTraffic))
	freeMetric.WithLabelValues(vpsID.VMId).Set(vm.VMInformation.Bandwidth.FreeTraffic)
	freePercentMetric.WithLabelValues(vpsID.VMId).Set(vm.VMInformation.Bandwidth.FreeTrafficPercent)
	usedMetric.WithLabelValues(vpsID.VMId).Set(vm.VMInformation.Bandwidth.UsedTraffic)
	usedPercentMetric.WithLabelValues(vpsID.VMId).Set(vm.VMInformation.Bandwidth.UsedTrafficPercent)
}

// checkEnvs checks required environment variables. If any of them are not defined, then it terminates application.
func checkEnvs() {
	if apiKey == "" {
		logrus.Error("Env SCALERA_API_KEY is not defined.")
		os.Exit(1)
	}
	if apiPassword == "" {
		logrus.Error("Env SCALERA_API_PASSWORD is not defined.")
		os.Exit(1)
	}
	if scaleraUrl == "" {
		logrus.Error("Env SCALERA_URL is not defined.")
		os.Exit(1)
	}
	if scrapeTimeEnv == "" {
		logrus.Warning("Env SCALERA_SCRAPE_SCHEDULE is not defined.")
		scrapeTime = 5
	} else {
		scrapeTime, _ = strconv.Atoi(scrapeTimeEnv)
	}
	if insecureSSLCheckStatus == "" {
		logrus.Warning("Env SCALERA_IGNORE_SSL is not defined. Default is FALSE.")
	} else {
		insecureSSLCheck, _ = strconv.ParseBool(insecureSSLCheckStatus)
	}
}
