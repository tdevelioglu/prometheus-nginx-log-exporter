package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/hpcloud/tail"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/satyrius/gonx"

	"github.com/tdevelioglu/prometheus-nginx-log-exporter/logging"
)

const version = "1.1.0"

var logger *logging.Logger

var reqParamsRegex = regexp.MustCompile(`\?.*`)
var staticLabels = []string{"method", "path", "status"}

func main() {
	if err := do_main(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func do_main() error {
	configFile := flag.String("config-file", "prometheus-nginx-log-exporter.yaml",
		"Path to a YAML file to read configuration from")
	showVersion := flag.Bool("version", false, "Prints the version and exits")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return nil
	}

	if *configFile == "" {
		return errors.New("Must specify config using -config-file")
	}

	var gc *globalConfig
	var err error
	if gc, err = newConfig(*configFile); err != nil {
		return err
	}

	if logger, err = logging.NewLogger(gc.LogLevel); err != nil {
		return err
	}

	for n := range gc.Applications {
		monitorApp(n, gc.Applications[n])
	}

	listenAddr := fmt.Sprintf("%s:%d", gc.Listen.Address, gc.Listen.Port)
	fmt.Printf("running HTTP server on %s\n", listenAddr)

	http.Handle("/metrics", promhttp.Handler())
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		return err
	}
	return nil
}

// monitorApp sets up the parsers and metrics for the log files
// that belong to a single application
func monitorApp(name string, ac *appConfig) {
	ln := append(staticLabels, ac.orderedLabelNames...)
	metrics := newMetrics(name, ln, ac.HistogramBuckets)

	parser := gonx.NewParser(ac.Format)
	for _, file := range ac.LogFiles {
		go monitorFile(file, parser, metrics, ac.orderedLabelValues, ac.Exclude, ac.Include, ac.Replace, ac.FromBeginning)
	}
}

// monitorFile tracks and collects metrics for a single log file.
func monitorFile(file string, parser *gonx.Parser, metrics *metrics, extraLabelValues []string,
	exclude []filterConfig, include []filterConfig, replace []replaceConfig, fromBeginning bool) {
	logger.Info("(%s): starting log file monitoring\n", file)

	// tail silently quits if the file exists but is inaccessible, so we test
	// here ourselves.
	f, err := os.Open(file)
	if err != nil {
		logger.Err("(%s): %s\n", file, err)
		os.Exit(1)
	} else {
		f.Close()
	}

	// We start from either the beginning or the end of the log file.
	var location *tail.SeekInfo
	if fromBeginning {
		location = nil
	} else {
		location = &tail.SeekInfo{Offset: 0, Whence: 2}
	}

	t, err := tail.TailFile(file, tail.Config{
		Follow:   true,
		ReOpen:   true,
		Location: location,
	})
	if err != nil {
		logger.Err("(%s): %s\n", file, err)
		os.Exit(1)
	}

	labelValues := make([]string, len(extraLabelValues)+len(staticLabels))
	for i := range extraLabelValues {
		labelValues[i+len(staticLabels)] = extraLabelValues[i]
	}

LINES:
	for line := range t.Lines {
		logger.Debug("(%s): parsing line: '%s'", file, line.Text)
		entry, err := parser.ParseString(line.Text)
		if err != nil {
			logger.Warn("(%s): failed to parse line: %s\n", file, err)
			continue
		}

		labelValues[0] = "" // method
		labelValues[1] = "" // path
		labelValues[2] = "" // status

		if request, err := entry.Field("request"); err == nil {
			method, path, err := parseRequest(request)
			if err != nil {
				logger.Warn("(%s): Failed to parse request field: %s", err)
				continue
			}
			logger.Debug("(%s): method and path are: %s and %s", file, method, path)

			if len(exclude) > 0 {
				if fc := filterMatch(method, path, exclude); fc != nil {
					logger.Debug("(%s): found matching exclude rule: '%s' %v",
						file, fc.pathRe.String(), fc.Methods)
					continue LINES
				} else {
					logger.Debug("(%s): no matching exclude rule found", file)
				}
			}

			if len(include) > 0 {
				if fc := filterMatch(method, path, include); fc != nil {
					logger.Debug("(%s): found matching include rule: '%s' %v",
						file, fc.pathRe.String(), fc.Methods)
				} else {
					logger.Debug("(%s): no matching include rule found", file)
					continue LINES
				}
			}

			if len(replace) > 0 {
				if rc := replaceMatch(method, path, replace); rc != nil {
					logger.Debug("(%s): found matching replace rule, %s %v -> %s",
						file, rc.pathRe.String(), rc.Methods, rc.With)
					if rc.useTemplate() {
						// use template to update path
						path, err = rc.replaceWithTemplate(path)
						if err != nil {
							logger.Warn("(%s): replace path (%s) with template (%s) error, %v", file, path, rc.With, err)
							continue
						}
					} else {
						// use regex to update path
						path = rc.replace(path)
					}
				} else {
					logger.Debug("(%s): no matching replace rule found", file)
				}
			}
			labelValues[0] = method
			labelValues[1] = path
			logger.Debug("(%s): matched path to %s", file, path)
		}

		if status, err := entry.Field("status"); err == nil {
			logger.Debug("(%s): matched status to %s", file, status)
			labelValues[2] = status
		}

		if bodyBytes, err := entry.FloatField("body_bytes_sent"); err == nil {
			logger.Debug("(%s): matched body_bytes_sent to %.f", file, bodyBytes)
			metrics.bodyBytes.WithLabelValues(labelValues...).Observe(bodyBytes)
		}

		if upstreamTime, err := entry.Field("upstream_response_time"); err == nil {
			if totalTime, err := parseUpstreamTime(upstreamTime); err == nil {
				logger.Debug("(%s): matched upstream_response_time to %.3f", file, totalTime)
				metrics.upstreamSeconds.WithLabelValues(labelValues...).Observe(totalTime)
			} else {
				logger.Warn("(%s): failed to parse upstream_response_time field", file, err)
			}
		}

		if headerTime, err := entry.Field("upstream_header_time"); err == nil {
			if totalTime, err := parseUpstreamTime(headerTime); err == nil {
				logger.Debug("(%s): matched upstream_header_time to %.3f", file, totalTime)
				metrics.upstreamHeaderSeconds.WithLabelValues(labelValues...).Observe(totalTime)
			} else {
				logger.Warn("(%s): failed to parse upstream_header_time field", file, err)
			}
		}

		if responseTime, err := entry.FloatField("request_time"); err == nil {
			logger.Debug("(%s): matched request_time to %.3f", file, responseTime)
			metrics.requestSeconds.WithLabelValues(labelValues...).Observe(responseTime)
		}

	}
}

// filterMatch checks if a method/path combination matches a list of filters.
func filterMatch(method string, path string, f []filterConfig) *filterConfig {
	for i := range f {
		if f[i].match(method, path) {
			return &f[i]
		}
	}
	return nil
}

// replaceMatch checks if a method/path combination matches a list of filters.
func replaceMatch(method string, path string, f []replaceConfig) *replaceConfig {
	for i := range f {
		if f[i].match(method, path) {
			return &f[i]
		}
	}
	return nil
}

// parseRequest parses an nginx $request value into method and path.
func parseRequest(request string) (string, string, error) {
	fields := strings.Split(request, " ")

	if len(fields) < 2 {
		return "", "", requestParseError(request)
	}

	path, err := url.PathUnescape(fields[1])
	if err != nil {
		return "", "", err
	}
	path = reqParamsRegex.ReplaceAllLiteralString(path, "")

	return fields[0], path, nil
}

// parseUpstreamTime sums an nginx $upstream_response_time value into a single float.
func parseUpstreamTime(upstreamTime string) (float64, error) {
	var totalTime float64

	for _, timeString := range strings.Split(upstreamTime, ", ") {
		time, err := strconv.ParseFloat(timeString, 32)
		if err != nil {
			return -1, err
		}
		totalTime = totalTime + time
	}
	return totalTime, nil
}
