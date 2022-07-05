# mget
![GitHub](https://img.shields.io/github/license/StevenCyb/mget)

Simple client request, parse and filter metrics from api endpoint.
This package is not production ready and will probably not be developed further.

## Just request metrics
```golang
endpoint := "https://somewhere.com/metrics"
result := mget.NewClient().Endpoint(endpoint).Do(context.Background())

/*
 * When the endpoint returns the following metrics:
 *    # HELP go_gc_duration_seconds A summary of the pause duration.
 *    # TYPE go_gc_duration_seconds summary
 *    go_gc_duration_seconds{quantile="0"} 3.275e-05
 *    go_gc_duration_seconds{quantile="0.5"} 0.000126252
 *    go_gc_duration_seconds{quantile="1"} 0.096255362
 *    go_gc_duration_seconds_count 7971
 *    # HELP go_goroutines Number of goroutines that currently exist.
 *    # TYPE go_goroutines gauge
 *    go_goroutines 86
 *    # HELP go_info Information about the Go environment.
 *    # TYPE go_info gauge
 *    go_info{version="go1.17.2"} 1
 * 
 * Then the result would look like:
 *    {
  *     "ResponseStatus":200,
  *     "Err":null,
  *     "Metric":[
  *         {
  *           "Name":"go_gc_duration_seconds",
  *           "Help":"A summary of the pause duration.",
  *           "Type":"SUMMARY",
  *           "TypeRaw":"summary",
  *           "Values":[
  *               { "Label":{"quantile":"0"}, "Value":0.00003275},
  *               { "Label":{"quantile":"0.5"}, "Value":0.000126252},
  *               { "Label":{"quantile":"1"}, "Value":0.096255362}
  *           ]
  *         },
  *         {
  *           "Name":"go_gc_duration_seconds_count",
  *           "Help":"",
  *           "Type":"UNKNOWN",
  *           "TypeRaw":"",
  *           "Values":[
  *               { "Label":{}, "Value":7971}
  *           ]
  *         },
  *         {
  *           "Name":"go_goroutines",
  *           "Help":"Number of goroutines that currently exist.",
  *           "Type":"GAUGE",
  *           "TypeRaw":"gauge",
  *           "Values":[
  *               { "Label":{}, "Value":86}
  *           ]
  *         },
  *         {
  *           "Name":"go_info",
  *           "Help":"Information about the Go environment.",
  *           "Type":"GAUGE",
  *           "TypeRaw":"gauge",
  *           "Values":[
  *               { "Label":{"version":"go1.17.2"}, "Value":1}
  *           ]
  *         },
  *     ]
  *   }
 */
```

## Filter 
### By name
```golang
result := mget.NewClient().
	Endpoint(endpoint).
	FilterByName("go_gc_duration_seconds_count", "go_goroutines").
	Do(context.Background())
```
### By metric type
```golang
result := mget.NewClient().
	Endpoint(endpoint).
	FilterByType(mget.SummaryType).
	Do(context.Background())
```
### By metric type
```golang
result := mget.NewClient().
	Endpoint(endpoint).
	FilterByType(mget.GaugeType).
	Do(context.Background())
```
### By metric labels
```golang
result := mget.NewClient().
	Endpoint(endpoint).
	FilterByLabel(map[string][]string{
		"service_name":    {"service_a", "service_b", "service_c"},
		"boundary": {"backend"},
	}).
	Do(context.Background())
```