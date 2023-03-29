package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Device struct {
	ID       int    `json:"id"`
	Mac      string `json:"mac"`
	Firmware string `json:"firmware"`
}

type metrics struct {
	devices prometheus.Gauge
	info    *prometheus.GaugeVec
}

func NewMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		devices: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "monitoringsystem",
			Name:      "connected_devices",
			Help:      "Number of connected devices",
		}),
		info: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "monitoringsystem",
			Name:      "info",
			Help:      "Info about the enviroment",
		},
			[]string{"version"},
		),
	}
	reg.MustRegister(m.devices, m.info)
	return m
}

var dvs []Device
var version string

func init() {
	version = "1.0.0"

	dvs = []Device{
		{ID: 1, Mac: "65:D0:E8:1A:26:EA", Firmware: "2.1.6"},
		{ID: 2, Mac: "65:D0:E9:2A:44:EB", Firmware: "1.1.2"},
	}
}

func main() {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.devices.Set(float64(len(dvs)))
	m.info.With(prometheus.Labels{"version": version}).Set(1)

	// http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	// http.HandleFunc("/devices", getDevices)
	// http.ListenAndServe(":8080", nil)

	dMux := http.NewServeMux()
	rdh := registerDeviceHandler{metrics: m}
	dMux.Handle("/devices", rdh)

	pMux := http.NewServeMux()
	pMux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	go func() {
		log.Fatal(http.ListenAndServe(":8080", dMux))
	}()

	go func() {
		log.Fatal(http.ListenAndServe(":8081", pMux))
	}()

	select {}
}

type registerDeviceHandler struct {
	metrics *metrics
}

func (rdh registerDeviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getDevices(w, r)
	case "POST":
		createDevice(w, r, rdh.metrics)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getDevices(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(dvs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func createDevice(w http.ResponseWriter, r *http.Request, m *metrics) {
	var dv Device

	err := json.NewDecoder(r.Body).Decode(&dv)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dvs = append(dvs, dv)
	m.devices.Set(float64(len(dvs)))

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Device created"))
}
