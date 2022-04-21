// 	file: check_cisco_uc_perf.go
// 	Version 0.8 (21.04.2021)
//
// check_cisco_uc_perf is a Nagios plugin made by Herwig Grimm (herwig.grimm at aon.at)
// to monitor the performance Cisco Unified Communications Servers.
//
// I have used the Google Go programming language because of no need to install
// any libraries.
//
// The plugin uses the Cisco PerfmonPort SOAP Service via HTTPS to do a wide variety of checks.
//
// This nagios plugin is free software, and comes with ABSOLUTELY NO WARRANTY.
// It may be used, redistributed and/or modified under the terms of the GNU
// General Public Licence (see http://www.fsf.org/licensing/licenses/gpl.txt).
//
// log files and cache file:
//  		befor first use create the following log files and cache file
//  		touch /var/log/check_cisco_uc_perf.log
//  		chown nagios.nagios /var/log/check_cisco_uc_perf.log
//
//  		mkdir /tmp/check_cisco_uc_perf_cache
//  		chown nagios.nagios  /tmp/check_cisco_uc_perf_cache
//
//
// tested with:
// 			Cisco Unified Communications Manager CUCM version 8.6.2.22900-9
//			Cisco Unified Communications Manager CUCM version 9.1.2.11900-12
//			Cisco Unified Communications Manager CUCM version 11.0.1.20000-2
//			Cisco Unified Communications Manager CUCM version 14
//
// see also:
// 		Cisco Unified Communications Manager XML Developers Guide, Release 9.0(1)
// 		http://www.cisco.com/c/en/us/td/docs/voice_ip_comm/cucm/devguide/9_0_1/xmldev-901.html
//
// changelog:
//		Version 0.1 (15.05.2014) initial release
//		Version 0.2 (20.05.2014) object caching added. new func loadStruct and saveStruct
//		Version 0.3 (27.02.2015) General Public Licence added
//		Version 0.3.1 (27.02.2015) new flag -m maximum cache age in seconds and flag -a and flag -A Cisco AXL API version of AXL XML Namespace
//		Version 0.3.2 (27.02.2015) changed flag -H usage description
//		Version 0.3.3 (30.11.2015) CUCM version 11.0: in TLSClientConfig MaxVersion set to tls.VersionTLS11 (TLS 1.1)
//		Version 0.4 (02.02.2016) new flag -M query multiple CUCM nodes.
//		Version 0.5 (12.03.2020) now first step: flag.Parse() and then check if logFileName is writeable
//		...
//		Version 0.8 (21.04.2021) XML data parsing largely reworked. New argument -C to define the cache file path and new argument -L to define the log filename.

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	outputPrefix     = "UC Perfmon"
	version          = "0.8"
	chacheFilePrefix = "check_cisco_uc_perf_"
)

type (
	PerfmonListCounter struct {
		XMLName struct{} `xml:"soap:perfmonListCounter"`
		Host    string   `xml:"soap:Host"`
	}

	PerfmonCollectCounterData struct {
		XMLName struct{} `xml:"soap:perfmonCollectCounterData"`
		Host    string   `xml:"soap:Host"`
		Object  string   `xml:"soap:Object"`
	}

	CounterEnvelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Text    string   `xml:",chardata"`
		Soapenv string   `xml:"soapenv,attr"`
		Xsd     string   `xml:"xsd,attr"`
		Xsi     string   `xml:"xsi,attr"`
		Body    struct {
			Text                              string `xml:",chardata"`
			PerfmonCollectCounterDataResponse struct {
				Text               string `xml:",chardata"`
				EncodingStyle      string `xml:"encodingStyle,attr"`
				Ns1                string `xml:"ns1,attr"`
				ArrayOfCounterInfo struct {
					Text               string `xml:",chardata"`
					ArrayType          string `xml:"arrayType,attr"`
					Type               string `xml:"type,attr"`
					Ns2                string `xml:"ns2,attr"`
					Soapenc            string `xml:"soapenc,attr"`
					ArrayOfCounterInfo []struct {
						Text string `xml:",chardata"`
						Type string `xml:"type,attr"`
						Name struct {
							Text string `xml:",chardata"`
							Type string `xml:"type,attr"`
						} `xml:"Name"`
						Value struct {
							Text string `xml:",chardata"`
							Type string `xml:"type,attr"`
						} `xml:"Value"`
						CStatus struct {
							Text string `xml:",chardata"`
							Type string `xml:"type,attr"`
						} `xml:"CStatus"`
					} `xml:"ArrayOfCounterInfo"`
				} `xml:"ArrayOfCounterInfo"`
			} `xml:"perfmonCollectCounterDataResponse"`
		} `xml:"Body"`
	}

	ListCounterEnvelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Text    string   `xml:",chardata"`
		Soapenv string   `xml:"soapenv,attr"`
		Xsd     string   `xml:"xsd,attr"`
		Xsi     string   `xml:"xsi,attr"`
		Body    struct {
			Text                       string `xml:",chardata"`
			PerfmonListCounterResponse struct {
				Text              string `xml:",chardata"`
				EncodingStyle     string `xml:"encodingStyle,attr"`
				Ns1               string `xml:"ns1,attr"`
				ArrayOfObjectInfo struct {
					Text              string `xml:",chardata"`
					ArrayType         string `xml:"arrayType,attr"`
					Type              string `xml:"type,attr"`
					Ns2               string `xml:"ns2,attr"`
					Soapenc           string `xml:"soapenc,attr"`
					ArrayOfObjectInfo []struct {
						Text string `xml:",chardata"`
						Type string `xml:"type,attr"`
						Name struct {
							Text string `xml:",chardata"`
							Type string `xml:"type,attr"`
						} `xml:"Name"`
						MultiInstance struct {
							Text string `xml:",chardata"`
							Type string `xml:"type,attr"`
						} `xml:"MultiInstance"`
						ArrayOfCounter struct {
							Text           string `xml:",chardata"`
							ArrayType      string `xml:"arrayType,attr"`
							Type           string `xml:"type,attr"`
							ArrayOfCounter []struct {
								Text string `xml:",chardata"`
								Type string `xml:"type,attr"`
								Name struct {
									Text string `xml:",chardata"`
									Type string `xml:"type,attr"`
								} `xml:"Name"`
							} `xml:"ArrayOfCounter"`
						} `xml:"ArrayOfCounter"`
					} `xml:"ArrayOfObjectInfo"`
				} `xml:"ArrayOfObjectInfo"`
			} `xml:"perfmonListCounterResponse"`
		} `xml:"Body"`
	}
)

var (
	ipAddr            string
	nodeIpAddr        string
	nodesIpAddrs      string
	username          string
	password          string
	objectInstance    string
	counterName       string
	debug             int
	warningThreshold  string
	criticalThreshold string
	showVersion       bool
	showCounters      bool
	maxCacheAge       int64
	apiVersion        string
	usePersistData    bool
	returnVal         int
	multipeNodes      bool
	logFileName       string
	cacheFilePath     string
)

func debugPrintf(level int, format string, a ...interface{}) {

	if level == 1 || level <= debug {
		log.Printf(format, a...)
	}
}

func isFullQualified(counterName string) bool {
	r, err := regexp.Compile(`^\\\\.*\\.*\\.*`)
	if err != nil {
		debugPrintf(1, "regexp compile error: %s\n", err)
		os.Exit(3)
	}
	if r.MatchString(counterName) {
		return true
	} else {
		return false
	}
}

// save struct to json file in tmp dir
func saveStruct(ipAddr, object string, o *CounterEnvelope) bool {

	itemJson, err := json.Marshal(o)
	if err != nil {
		debugPrintf(1, "error: %s", err)
		return false
	}

	objectUnderscore := strings.Replace(object, " ", "_", -1)
	filename := fmt.Sprintf("%s%s%d_%s_%s", cacheFilePath, chacheFilePrefix, os.Getuid(), ipAddr, objectUnderscore)

	err = ioutil.WriteFile(filename, itemJson, 0666)

	if err != nil {
		debugPrintf(1, "error: %s", err)
		return false
	}

	return true
}

// load struct from json file in tmp dir if newer than defined in ageInSeconds
func loadStruct(ipAddr, object string, ageInSeconds int64, o *CounterEnvelope) bool {

	objectUnderscore := strings.Replace(object, " ", "_", -1)
	filename := fmt.Sprintf("%s%s%d_%s_%s", cacheFilePath, chacheFilePrefix, os.Getuid(), ipAddr, objectUnderscore)

	fs, err := os.Stat(filename)
	if err != nil {
		// debugPrintf(1, "error 1: %s", err)
		return false
	}

	debugPrintf(3, "Filename: %s Diff: %d\n", filename, time.Now().Unix()-fs.ModTime().Unix())
	if (time.Now().Unix() - fs.ModTime().Unix()) > ageInSeconds {
		return false
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		debugPrintf(1, "error: %s", err)
		return false
	}
	err = json.Unmarshal(data, &o)
	if err != nil {
		debugPrintf(1, "error: %s", err)
		return false
	}
	return true
}

// Determine plugin return codes based threshold ranges
// according to "Nagios Plugin Development Guidelines"
// section "Plugin Return Codes, Threshold and ranges"
// see https://nagios-plugins.org/doc/guidelines.html
func getNagiosReturnVal(value float64, warningThresholdRange, criticalThresholdRange string) int {
	r := 0
	if generateAlert(value, warningThresholdRange) {
		r = 1 // warning
	}
	if generateAlert(value, criticalThresholdRange) {
		r = 2 // critical
	}
	return r
}

// Match value against threshold range
// according to "Nagios Plugin Development Guidelines"
// section "Plugin Return Codes, Threshold and ranges"
// see https://nagios-plugins.org/doc/guidelines.html
func generateAlert(value float64, thresholdRange string) bool {
	r := strings.Split(thresholdRange, ":")
	matched, _ := regexp.MatchString(`^[0-9.]+:[0-9.]+`, thresholdRange)
	switch {
	case len(r) == 1:
		float64_threshold, _ := strconv.ParseFloat(thresholdRange, 64)
		return value < 0 || value > float64_threshold
	case strings.HasSuffix(thresholdRange, ":"):
		float64_threshold, _ := strconv.ParseFloat(r[0], 64)
		return value < float64_threshold
	case strings.HasPrefix(thresholdRange, "~"):
		float64_threshold, _ := strconv.ParseFloat(r[1], 64)
		return value > float64_threshold
	case matched:
		float64_threshold1, _ := strconv.ParseFloat(r[0], 64)
		float64_threshold2, _ := strconv.ParseFloat(r[1], 64)
		return value < float64_threshold1 || value > float64_threshold2
	case strings.HasPrefix(thresholdRange, "@"):
		float64_threshold1, _ := strconv.ParseFloat(strings.TrimPrefix(r[0], "@"), 64)
		float64_threshold2, _ := strconv.ParseFloat(r[1], 64)
		return value >= float64_threshold1 && value <= float64_threshold2
	}
	return true
}

func returnValText(returnVal int) string {
	statusStr := ""
	switch returnVal {
	case 0:
		statusStr = "OK"
	case 1:
		statusStr = "WARNING"
	case 2:
		statusStr = "CRITICAL"
	case 3:
		statusStr = "UNKNOWN"
	default:
		statusStr = ""
	}
	return statusStr
}

func init() {
	flag.StringVar(&ipAddr, "H", "", "CUCM server IP address")
	flag.StringVar(&nodeIpAddr, "N", "", "Node IP address")
	flag.StringVar(&nodesIpAddrs, "M", "", "Comma separated list of nodes (IP addresses)")
	flag.StringVar(&username, "u", "", "username")
	flag.StringVar(&password, "p", "", "password")
	flag.StringVar(&objectInstance, "o", "Memory", "Perfmon object with optional tailing instance names in parenthesis")
	flag.StringVar(&counterName, "n", "", "Counter name")
	flag.IntVar(&debug, "d", 0, "print debug, level: 1 errors only, 2 warnings and 3 informational messages")
	flag.StringVar(&warningThreshold, "w", "1", "Warning threshold or threshold range")
	flag.StringVar(&criticalThreshold, "c", "1", "Critical threshold or threshold range")
	flag.BoolVar(&showVersion, "V", false, "print plugin version")
	flag.BoolVar(&showCounters, "l", false, "print PerfmonListCounter")
	flag.Int64Var(&maxCacheAge, "m", 180, "maximum cache age in seconds")
	flag.StringVar(&apiVersion, "A", "9.0", "Cisco AXL API version of AXL XML Namespace")
	flag.StringVar(&logFileName, "L", "/var/log/check_cisco_uc_perf.log", "Log file path and name")
	flag.StringVar(&cacheFilePath, "C", "/tmp/check_cisco_uc_perf/", "Cache file path")
}

func queryHost(ipAddr, nodeIpAddr, object, counterName, objectInstance string) {

	fullCounterName := ""

	debugPrintf(3, "queryHost CUCM IP address: %s Node IP address: %s\n", ipAddr, nodeIpAddr)
	debugPrintf(3, "queryHost perfmon object: %s Counter name: %s\n", object, counterName)
	debugPrintf(3, "queryHost counter instance name: %s max cache age: %d\n", objectInstance, maxCacheAge)

	counterEnvelope := new(CounterEnvelope)
	loaded := loadStruct(nodeIpAddr, object, maxCacheAge, counterEnvelope)
	if !loaded {
		debugPrintf(3, "No persistence file found or persistence file too old\n")
		usePersistData = false
	} else {
		debugPrintf(3, "Persistence file found: %+v\n", counterEnvelope)
		if isFullQualified(counterName) {
			fullCounterName = counterName
		} else {
			fullCounterName = fmt.Sprintf("\\\\%s\\%s\\%s", nodeIpAddr, object, counterName)
		}
		for _, v := range counterEnvelope.Body.PerfmonCollectCounterDataResponse.ArrayOfCounterInfo.ArrayOfCounterInfo {
			if v.Name.Text == fullCounterName {
				debugPrintf(3, "Name: %s Value: %s\n", v.Name.Text, v.Value.Text)
			}
		}
		usePersistData = true
	}

	debugPrintf(3, "use persistence: %v\n", usePersistData)
	if !usePersistData || showCounters {

		client := &http.Client{

			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
					MaxVersion:         tls.VersionTLS11,
				},
			},
		}

		xml_header := []byte(`<?xml version="1.0" encoding="utf-8" ?><soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:soap="http://schemas.cisco.com/ast/soap"><soapenv:Header/><soapenv:Body>`)
		xml_footer := []byte(`</soapenv:Body></soapenv:Envelope>`)

		xml_data := make([]byte, 32768)

		if showCounters {
			req_data := &PerfmonListCounter{Host: nodeIpAddr}
			xml_data, _ = xml.Marshal(req_data)
		} else {
			req_data := &PerfmonCollectCounterData{Host: nodeIpAddr, Object: object}
			xml_data, _ = xml.Marshal(req_data)
		}

		buf_all := make([]byte, 32768)

		buf_all = append(buf_all, xml_header...)
		buf_all = append(buf_all, xml_data...)
		buf_all = append(buf_all, xml_footer...)

		xml_all := fmt.Sprintf("%s%s%s", xml_header, xml_data, xml_footer)

		debugPrintf(3, "XML SOAP request: %s\n", xml_all)

		data := bytes.NewBufferString(xml_all)

		url := "https://" + ipAddr + ":8443/perfmonservice/services/PerfmonPort"
		debugPrintf(3, "URL: %s\n", url)
		req, err := http.NewRequest("POST", url, data)
		req.Header.Add("Content-type", "text/xml")
		req.Header.Add("SOAPAction", "CUCM:DB ver="+apiVersion)
		req.SetBasicAuth(username, password)

		debugPrintf(3, "username: %s, password: %s\n", username, password)

		resp, err := client.Do(req)
		if err != nil {
			debugPrintf(1, "HTTPS request error: %s %#v\n", err, resp)
			os.Exit(3)
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)

		debugPrintf(3, "XML SOAP response: %s\n", body)

		if showCounters {

			listCounterEnvelope := new(ListCounterEnvelope)
			err = xml.Unmarshal([]byte(body), listCounterEnvelope)
			if err != nil {
				debugPrintf(1, "ListCounterEnvelope XML unmarshal error: %s\n", err)
				os.Exit(3)
			}

			debugPrintf(3, "PerfmonListCounterData: %+v\n", listCounterEnvelope.Body)

			fmt.Printf("%d items\n", len(listCounterEnvelope.Body.PerfmonListCounterResponse.ArrayOfObjectInfo.ArrayOfObjectInfo))

			for _, v := range listCounterEnvelope.Body.PerfmonListCounterResponse.ArrayOfObjectInfo.ArrayOfObjectInfo {
				fmt.Printf("%v\n", v.Name.Text)
				for _, c := range v.ArrayOfCounter.ArrayOfCounter {
					fmt.Printf("\t%s\n", c.Name.Text)
				}
			}

			os.Exit(0)
		}

		counterEnvelope = new(CounterEnvelope)
		err = xml.Unmarshal([]byte(body), counterEnvelope)
		if err != nil {
			debugPrintf(1, "XML unmarshal error: %s\n", err)
			os.Exit(3)
		}
		saveStruct(nodeIpAddr, object, counterEnvelope)

	}

	if len(counterName) > 0 {
		if isFullQualified(counterName) {
			fullCounterName = counterName
		} else {
			fullCounterName = fmt.Sprintf("\\\\%s\\%s\\%s", nodeIpAddr, objectInstance, counterName)
		}
		debugPrintf(3, "fullCounterName: >>%s<<\n", fullCounterName)
		debugPrintf(3, "envelope.Body.perfmonCollectCounterDataResponse: %+v\n", counterEnvelope)

		for _, v := range counterEnvelope.Body.PerfmonCollectCounterDataResponse.ArrayOfCounterInfo.ArrayOfCounterInfo {
			if v.Name.Text == fullCounterName {

				value, err := strconv.ParseFloat(v.Value.Text, 64)
				if err != nil {
					debugPrintf(1, "Counter value string to float64 convert error: %s\n", err)
					os.Exit(3)
				}
				returnVal = getNagiosReturnVal(value, warningThreshold, criticalThreshold)
				debugPrintf(3, "returnVal: %d\n", returnVal)
				statusStr := returnValText(returnVal)

				nagiosOutput := fmt.Sprintf("%s - %s,%s,%s=%s|%s=%s;%s;%s;;", statusStr, outputPrefix, objectInstance, counterName, v.Value.Text, counterName, v.Value.Text, warningThreshold, criticalThreshold)
				nagiosOutput = html.EscapeString(nagiosOutput)
				nagiosOutput = strings.Replace(nagiosOutput, "%", "Percent", -1)
				nagiosOutput = strings.Replace(nagiosOutput, "\\", "\\\\", -1)
				fmt.Printf("%s\n", nagiosOutput)
				os.Exit(returnVal)
			}
		}
		returnVal := 3
		statusStr := returnValText(returnVal)
		if multipeNodes {
			debugPrintf(3, "%s - Counter not found: %s\n", statusStr, fullCounterName)
		} else {
			fmt.Printf("%s - Counter not found: %s\n", statusStr, fullCounterName)
			os.Exit(returnVal)
		}

	}

}

func main() {

	flag.Parse()

	logfile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		debugPrintf(1, fmt.Sprintf("Can't open log file: %s\n", logFileName))
		os.Exit(3)
	}

	defer logfile.Close()

	returnVal = 3
	multipeNodes = false
	usePersistData = false

	if showVersion {
		fmt.Printf("%s version: %s\n", path.Base(os.Args[0]), version)
		os.Exit(0)
	}

	log.SetOutput(os.Stdout)

	// log.SetOutput(logfile)

	// remove tailing instance names and parenthesis
	object := ""
	if pos := strings.Index(objectInstance, "("); pos != -1 {
		object = objectInstance[:pos]
	} else {
		object = objectInstance
	}

	nodes := strings.Split(nodesIpAddrs, ",")

	if len(nodes) > 1 {
		multipeNodes = true
		debugPrintf(3, "multiple nodes: %v\n", nodes)
	}

	debugPrintf(3, "use multipe nodes: %v\n", multipeNodes)

	if multipeNodes {
		for _, nodeIpAddr = range nodes {
			queryHost(ipAddr, nodeIpAddr, object, counterName, objectInstance)
		}
	} else {
		queryHost(ipAddr, nodeIpAddr, object, counterName, objectInstance)
	}

}

