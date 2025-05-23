package kibana

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/atoscerebro/bms-analysis/internal/similarity"
	"github.com/go-viper/mapstructure/v2"
	"golang.org/x/sync/errgroup"
)

var WatcherErrorMapping = map[string][]string{
	"BMS_PRD1_DAILY_STATS":                        {},
	"BMS_PRD1_DispatcherDisabled":                 {},
	"BMS_PRD1_Err201Received":                     {"Err201Received"},
	"BMS_PRD1_FailedAuthenticating":               {"FailedAuthenticating"},
	"BMS_PRD1_FailedCallingBESS":                  {"ErrorCallingBESS", "ReceivedBESSFailureResponse"},
	"BMS_PRD1_FailedCallingBMSComponent":          {"ErrorCallingBMSComponent", "ReceivedBMSComponentFailureResponse"},
	"BMS_PRD1_FailedCallingBSGComponent":          {"ErrorCallingBSG", "ReceivedBSGFailureResponse"},
	"BMS_PRD1_FailedCallingDataPlatform":          {"ErrorCallingDataPlatform", "ReceivedDataPlatformFailureResponse"},
	"BMS_PRD1_FailedCallingSRTP":                  {"ErrorCallingSRTP", "ReceivedSRTPFailureResponse"},
	"BMS_PRD1_FailedChangingSQSVisibilityTimeout": {"FailedChangingSQSVisibilityTimeout"},
	"BMS_PRD1_FailedDeletingFromS3":               {"FailedDeletingFromS3"},
	"BMS_PRD1_FailedDeletingFromSQS":              {"FailedDeletingFromSQS"},
	"BMS_PRD1_FailedDeterminingRoute":             {"FailedDeterminingRoute"},
	"BMS_PRD1_FailedReceivingFromSQS":             {"FailedReceivingFromSQS"},
	"BMS_PRD1_FailedRetrievingFromS3":             {"FailedRetrievingFromS3"},
	"BMS_PRD1_FailedSendingToSQS":                 {"FailedSendingToSQS"},
	"BMS_PRD1_FailedSigning":                      {"FailedSigning"},
	"BMS_PRD1_FailedTransforming":                 {"FailedTransforming"},
	"BMS_PRD1_FailedValidating":                   {"FailedValidating"},
	"BMS_PRD1_GeneralError":                       {},
	"BMS_PRD1_ReceivedGrayScaleImage":             {},
	"BMS_PRD1_UnexpectedError":                    {"UnexpectedError"},
	"BMS_SUPPORT_TST1_FailedCallingBMSComponent":  {"ErrorCallingBMSComponent", "ReceivedBMSComponentFailureResponse"},
}

var AlertsWatcherOutputPath = "alerts-watcher-output.json"
var AlertsCoordinatesOutputPath = "alerts-coordinate-output.json"

type KibanaWatcherLogResult struct {
	ExecutionTime string `mapstructure:"execution_time" json:"execution_time"`
}

type KibanaWatcherLogSource struct {
	Result  KibanaWatcherLogResult `mapstructure:"result" json:"result"`
	WatchId string                 `mapstructure:"watch_id" json:"watch_id"`
}

type KibanaWatcherLog struct {
	ID          string                 `json:"_id"`
	Source      KibanaWatcherLogSource `json:"_source"`
	Sort        []interface{}          `json:"sort"`
	Coordinates KibanaLogCoordinates   `json:"coordinates"`
}

type KibanaWatcherLogs []*KibanaWatcherLog

func (c *KibanaClient) GetWatcherExecutions() (*KibanaWatcherLogs, error) {
	query := map[string]interface{}{
		"sort": []map[string]interface{}{
			{
				"result.execution_time": map[string]string{
					"order": "desc",
				},
			},
		},
		"_source": []string{
			"watch_id",
			"result.execution_time",
			"result.actions",
			"result.condition",
			"result.status",
		},
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"result.condition.met": true,
						},
					},
					{
						"prefix": map[string]interface{}{
							"watch_id": "BMS_",
						},
					},
					{
						"range": map[string]interface{}{
							"result.execution_time": map[string]string{
								"gte": "now-1M/M",
							},
						},
					},
				},
			},
		},
	}

	hits, err := c.SearchAll(".watcher-history-*", query)
	if err != nil {
		return nil, err
	}

	watcherHits := KibanaWatcherLogs{}
	for _, hit := range *hits {
		wl := KibanaWatcherLogSource{}
		if err := mapstructure.Decode(hit.Source, &wl); err != nil {
			return nil, fmt.Errorf("failed to convert log source to watcher source: %e", err)
		}
		watcherHits = append(watcherHits, &KibanaWatcherLog{
			ID:          hit.ID,
			Source:      wl,
			Sort:        hit.Sort,
			Coordinates: hit.Coordinates,
		})
	}

	return &watcherHits, nil
}

func (c *KibanaClient) GetWatcherErrorLogs(wlogs *KibanaWatcherLogs) (*KibanaErrorLogs, error) {
	mu := sync.Mutex{}
	results := KibanaErrorLogs{}
	g := new(errgroup.Group)
	g.SetLimit(runtime.NumCPU())
	for _, wl := range *wlogs {
		g.Go(func() error {
			defer func() {
				lenResults := len(results)
				if lenResults%(len(*wlogs)/10) == 0 {
					mu.Lock()
					log.Printf("fetched %d watcher error logs...", lenResults)
					mu.Unlock()
				}
			}()
			wlExecutionTime, err := time.Parse("2006-01-02T15:04:05.000Z", wl.Source.Result.ExecutionTime)
			if err != nil {
				return fmt.Errorf("failed to parse time: %e", err)
			}
			notFoundLog := &KibanaErrorLog{
				ID: wl.ID,
				Source: KibanaErrorLogSource{
					CorrelationId: wl.ID,
					Environment:   "prd1",
					HttpStatus:    0,
					Microservice:  "unknown",
					Message:       wl.Source.WatchId,
					ErrorMessage:  wl.Source.WatchId,
					TimeStamp:     wlExecutionTime.Format("2006-01-02T15:04:05.000Z07:00"),
				},
			}
			codes, exists := WatcherErrorMapping[wl.Source.WatchId]
			if !exists || len(codes) == 0 {
				mu.Lock()
				results = append(results, notFoundLog)
				mu.Unlock()
				return nil
			}

			wlExecutionTimeMs := wlExecutionTime.UnixMilli()
			query := map[string]interface{}{
				"size": 1,
				"sort": []map[string]interface{}{
					{
						"@timestamp": map[string]string{
							"order": "asc",
						},
					},
				},
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"must": []interface{}{
							map[string]interface{}{
								"terms": map[string][]string{
									"message.keyword": codes,
								},
							},
							map[string]interface{}{
								"range": map[string]interface{}{
									"@timestamp": map[string]interface{}{
										"gte":    wlExecutionTimeMs - 60000*10,
										"lte":    wlExecutionTimeMs + 60000*10,
										"format": "epoch_millis",
									},
								},
							},
						},
					},
				},
			}
			hits, err := c.Search("bms-*", query)
			if err != nil {
				return err
			}
			if len(hits.Hits.Hits) == 0 {
				mu.Lock()
				results = append(results, notFoundLog)
				mu.Unlock()
				return nil
			}
			var e KibanaErrorLogSource
			for i, hit := range hits.Hits.Hits {
				el := KibanaErrorLogSource{}
				if err := mapstructure.Decode(hit.Source, &el); err != nil {
					return fmt.Errorf("failed to convert log source to error source: %e", err)
				}
				if el.ErrorMessage != "" || i == len(hits.Hits.Hits)-1 {
					e = el
					break
				}
			}
			elTimestamp, err := time.Parse("2006-01-02T15:04:05.000Z", e.TimeStamp)
			if err != nil {
				return fmt.Errorf("failed to parse time: %e", err)
			}
			e.TimeStamp = elTimestamp.Format("2006-01-02T15:04:05.000Z07:00")
			hit := hits.Hits.Hits[0]
			mu.Lock()
			results = append(results, &KibanaErrorLog{
				ID:          wl.ID,
				Source:      e,
				Sort:        hit.Sort,
				Coordinates: hit.Coordinates,
			})
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return &results, nil
}

func (c *KibanaClient) AnalyseAlerts() error {
	var watcherErrorLogs *KibanaErrorLogs
	var err error

	watcherLogsFile, err := os.ReadFile(AlertsWatcherOutputPath)
	if err == nil {
		log.Println("loading logs from local file...")
		if err = json.Unmarshal(watcherLogsFile, &watcherErrorLogs); err != nil {
			return fmt.Errorf("failed to unmarshal logs file: %s", err)
		}
	} else {
		var watcherLogs *KibanaWatcherLogs

		log.Println("fetching watcher logs from kibana...")
		if watcherLogs, err = c.GetWatcherExecutions(); err != nil {
			return fmt.Errorf("failed to get logs: %s", err)
		}
		log.Println("fetching watcher error logs from kibana...")
		if watcherErrorLogs, err = c.GetWatcherErrorLogs(watcherLogs); err != nil {
			return fmt.Errorf("failed to get logs: %s", err)
		}
		log.Println("writing watcher error logs to local file...")
		if err = output(watcherErrorLogs, AlertsWatcherOutputPath); err != nil {
			return fmt.Errorf("failed to write logs: %s", err)
		}
	}

	log.Println("calculating alert similarity...")
	comparableLogs := make([]similarity.Comparable, len(*watcherErrorLogs))
	for i, l := range *watcherErrorLogs {
		comparableLogs[i] = &KibanaLogErrorComparable{l}
	}
	coords, err := similarity.Coordinates(comparableLogs)
	if err != nil {
		return fmt.Errorf("failed to calculate coordinates: %s", err)
	}
	for i, coord := range coords {
		(*watcherErrorLogs)[i].Coordinates.Error = coord
	}
	if err := output(watcherErrorLogs, AlertsCoordinatesOutputPath); err != nil {
		return fmt.Errorf("failed to write coordinates: %s", err)
	}

	return nil
}
