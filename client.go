package mget

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

// Client represents the client that holds
// request specifications and logic.
type Client struct {
	httpClient  *http.Client
	endpoint    string
	nameFilter  []string
	typeFilter  []MetricType
	labelFilter []map[string][]string
}

// HttpClient use given http client to perform request (e.g. with authentication)
func (client *Client) HttpClient(httpClient http.Client) *Client {
	client.httpClient = &httpClient
	return client
}

// Endpoint endpoint that will be requested, e.g. `http://localhost:8080/metrics`
func (client *Client) Endpoint(endpoint string) *Client {
	client.endpoint = endpoint
	return client
}

// FilterByName specifies a list of metric names relevant for the result result.
// Multiple filters are interpreted as `logical or`.
func (client *Client) FilterByName(filters ...string) *Client {
	if client.nameFilter == nil {
		client.nameFilter = filters
		return client
	}

	client.nameFilter = append(client.nameFilter, filters...)
	return client
}

// FilterByType specifies a list of metric types relevant for the result result.
// Multiple filters are interpreted as `logical or`.
func (client *Client) FilterByType(filters ...MetricType) *Client {
	if client.nameFilter == nil {
		client.typeFilter = filters
		return client
	}

	client.typeFilter = append(client.typeFilter, filters...)
	return client
}

// FilterByLabel specifies a list of metric labels relevant for the result result.
// Multiple filters are interpreted as `logical or`, each map defines a set that are
// interpreted as `logical and` where the arrays for an given key is also a `logical or`.
func (client *Client) FilterByLabel(filters ...map[string][]string) *Client {
	if client.labelFilter == nil {
		client.labelFilter = filters
		return client
	}

	client.labelFilter = append(client.labelFilter, filters...)
	return client
}

// Do executes the request and apply filter if specified
func (client *Client) Do(ctx context.Context) Result {
	// Perform request
	result := Result{}
	if client.endpoint == "" {
		result.Err = ErrEndpointMissing
		return result
	}

	req, err := http.NewRequest("GET", client.endpoint, nil)
	if err != nil {
		result.Err = err
		return result
	}
	req = req.WithContext(ctx)

	if client.httpClient == nil {
		client.httpClient = http.DefaultClient
	}
	resp, err := client.httpClient.Do(req)
	if err != nil {
		result.Err = err
		return result
	}
	result.ResponseStatus = resp.StatusCode
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		result.Err = err
		return result
	}

	// Parse response
	result.Metric = []Metric{}
	lastHelp := ""
	lastTypeRaw := ""
	lastType := UnknownType
	lastMetricName := ""

	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		if strings.HasPrefix(line, "# HELP ") {
			lastHelp = strings.ReplaceAll(line, "# HELP ", "")
			continue
		}
		if strings.HasPrefix(line, "# TYPE ") {
			lastTypeRaw = strings.ReplaceAll(line, "# TYPE ", "")
			line := strings.ToUpper(line)
			if strings.Contains(line, string(CounterType)) {
				lastType = CounterType
			}
			if strings.Contains(line, string(GaugeType)) {
				lastType = GaugeType
			}
			if strings.Contains(line, string(HistogramType)) {
				lastType = HistogramType
			}
			if strings.Contains(line, string(SummaryType)) {
				lastType = SummaryType
			}
			continue
		}

		regex := regexp.MustCompile(
			`^(?P<name>[a-zA-Z_:][a-zA-Z0-9_:]*)({(?P<labels>[^}]*)})? (?P<value>[0-9e.+-]*)$`)
		matches := regex.FindStringSubmatch(line)
		subMatchMap := map[string]string{}
		for i, name := range regex.SubexpNames() {
			if name != "" && len(matches[i]) > 0 {
				// Map name, value and labels available
				subMatchMap[name] = matches[i]
			}
		}
		if len(matches) != 5 && len(matches) != 3 {
			//              0| 1  | | 2 -  3 |  |  3   |
			// Match = 3 => ((name) (value))
			// Match = 4 => ((name)({(label)}) (value))
			result.Err = fmt.Errorf("invalid format detected %s => [%+v]", line, subMatchMap)
			return result
		}

		// Extract information from single metric
		metricName := subMatchMap["name"]
		value, err := strconv.ParseFloat(subMatchMap["value"], 64)
		if err != nil {
			result.Err = err
			return result
		}
		labels := map[string]string{}
		if labelsString, has := subMatchMap["labels"]; has {
			labelPairs := strings.Split(labelsString, ",")
			for _, label := range labelPairs {
				tmp := strings.Split(label, "=")
				labels[tmp[0]] = strings.ReplaceAll(tmp[1], "\"", "")
			}
		}
		if !strings.HasPrefix(lastHelp, metricName) {
			lastHelp = ""
		}
		if !strings.HasPrefix(lastTypeRaw, metricName) {
			lastTypeRaw = ""
			lastType = UnknownType
		}

		// apply filters
		if len(client.nameFilter) > 0 &&
			!slices.Contains(client.nameFilter, metricName) {
			continue
		}
		if len(client.typeFilter) > 0 &&
			!slices.Contains(client.typeFilter, lastType) {
			continue
		}
		if len(client.labelFilter) > 0 {
			matchAnySet := false
			for _, labelFilter := range client.labelFilter {
				matchSet := true
				for filterKey, filterValue := range labelFilter {
					value, has := labels[filterKey]
					if !has || !slices.Contains(filterValue, value) {
						matchSet = false
						break
					}
				}

				if matchSet {
					matchAnySet = true
					break
				}
			}
			if !matchAnySet {
				continue
			}
		}

		// Apply metrics to result
		if lastMetricName != metricName {
			lastMetricName = metricName
			result.Metric = append(result.Metric, Metric{
				Name:    metricName,
				Help:    strings.ReplaceAll(lastHelp, metricName+" ", ""),
				Type:    lastType,
				TypeRaw: strings.ReplaceAll(lastTypeRaw, metricName+" ", ""),
				Values: []LabelValuePair{
					{
						Label: labels,
						Value: value,
					},
				},
			})
			continue
		}
		result.Metric[len(result.Metric)-1].Values =
			append(result.Metric[len(result.Metric)-1].Values,
				LabelValuePair{
					Label: labels,
					Value: value,
				})
	}

	return result
}

// NewClient creates a new client instance
func NewClient() *Client {
	return &Client{}
}
