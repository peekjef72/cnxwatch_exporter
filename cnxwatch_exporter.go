// Copyright 2019 David de Torres
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"net/http"
	"os"

	// "strings"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	// Constant values
	metricsPublishingPort = ":9293"
)

var (
	listenAddress = kingpin.Flag("web.listen-address", "The address to listen on for HTTP requests.").Default(metricsPublishingPort).String()
	metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose collector's internal metrics.").Default("/metrics").String()
	configFile    = kingpin.Flag("config-file", "Exporter configuration file.").Short('c').Default("config/config.yml").String()
	dry_run       = kingpin.Flag("dry-run", "Only check exporter configuration file and exit.").Short('n').Default("false").Bool()
	debug_flag    = kingpin.Flag("debug", "debug connection checks.").Short('d').Default("false").Bool()
)

//***********************************************************************************************
func handler(w http.ResponseWriter, r *http.Request, exporter *SocketSetExporter) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(exporter)
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

//***********************************************************************************************
func main() {
	var sockets *socketSet
	var err error

	logConfig := promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, &logConfig)
	kingpin.Version(version.Print("cnxwatch_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	// Setup build info metric.
	// version.Branch = Branch
	// version.BuildDate = BuildDate
	// version.Revision = Revision
	// version.Version = VersionShort

	logger := promlog.New(&logConfig)
	level.Info(logger).Log("msg", "Starting cnxwatch_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())

	// read the configuration if not empty
	if *configFile != "" {
		sockets, err = Load(*configFile)
		if err != nil {
			level.Error(logger).Log("Errmsg", "Error loading config", "err", err)
			os.Exit(1)
		}
	}
	if *dry_run {
		level.Info(logger).Log("msg", "configuration OK.")
		// os.Exit(0)
	}
	// create a new exporter
	sockExporter := NewSocketSetExporter(sockets, logger, *debug_flag)
	//	prometheus.MustRegister(sockExporter)
	level.Info(logger).Log("msg", "Connection Watch Exporter initialized")
	if *dry_run {
		level.Info(logger).Log("msg", "Connection Watch Exporter runs once in dry-mode (output to stdout).")
		registry := prometheus.NewRegistry()
		registry.MustRegister(sockExporter)
		mfs, err := registry.Gather()
		if err != nil {
			level.Error(logger).Log("Errmsg", "Error gathering metrics", "err", err)
			os.Exit(1)
		}
		enc := expfmt.NewEncoder(os.Stdout, expfmt.FmtText)

		for _, mf := range mfs {
			err := enc.Encode(mf)
			if err != nil {
				level.Error(logger).Log("Errmsg", err)
				break
			}
		}
		if closer, ok := enc.(expfmt.Closer); ok {
			// This in particular takes care of the final "# EOF\n" line for OpenMetrics.
			closer.Close()
		}
		os.Exit(1)
	}

	var landingPage = []byte(`<html>
		<head>
		<title>Connection Watch Exporter</title>
		</head>
		<body>
		<h1>Connection Watch Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
		</body>
		</html>
	`)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8") // nolint: errcheck
		w.Write(landingPage)                                       // nolint: errcheck
	})

	http.HandleFunc(*metricsPath, func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, sockExporter)
	})

	level.Info(logger).Log("msg", "Listening on address", "address", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		level.Error(logger).Log("msg", "Error starting HTTP server")
		os.Exit(1)
	}
}
