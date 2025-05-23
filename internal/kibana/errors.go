package kibana

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/atoscerebro/bms-analysis/internal/similarity"
	"github.com/go-viper/mapstructure/v2"
)

var ErrorKeywords = []string{
	"ErrorCallingBMSComponent",
	"ErrorCallingBSG",
	"ErrorCallingDataPlatform",
	"ErrorCallingRedHatSSO",
	"ErrorCallingSRTP",
	"FailedAuthenticating",
	"FailedChangingSQSVisibilityTimeout",
	"FailedDeletingFromSQS",
	"FailedDeterminingRoute",
	"FailedReceivingFromSQS",
	"FailedSendingToSQS",
	"FailedTransforming",
	"FailedValidating",
	"UnexpectedError",
	"ErrorCallingBESS",
	"FailedSigning",
	"FailedWritingToS3",
	"FailedRetrievingFromS3",
	"FailedDeletingFromS3",
	"Err201Received",
}

var ErrorsMessageOutputPath = "errors-message-output.json"
var ErrorsCoordinatesOutputPath = "errors-coordinate-output.json"

type KibanaErrorLogSource struct {
	CorrelationId string `json:"correlationId"`
	TCR           string `json:"tcr"`
	Environment   string `json:"environment"`
	HttpStatus    int32  `json:"httpStatus"`
	Message       string `json:"message"`
	Microservice  string `json:"microservice"`
	ErrorMessage  string `json:"errorMessage"`
	TimeStamp     string `json:"@timestamp"`
}

type KibanaErrorLog = struct {
	ID          string               `json:"_id"`
	Source      KibanaErrorLogSource `json:"_source"`
	Sort        []interface{}        `json:"sort"`
	Coordinates KibanaLogCoordinates `json:"coordinates"`
}

type KibanaLogErrorComparable struct {
	*KibanaErrorLog
}

func (kl *KibanaLogErrorComparable) Metric() string {
	return kl.Source.ErrorMessage
}

type KibanaErrorLogs []*KibanaErrorLog

func (kl *KibanaErrorLogs) ByMessage() map[string][]*KibanaErrorLog {
	result := map[string][]*KibanaErrorLog{}
	for _, hit := range *kl {
		_, ok := result[hit.Source.Message]
		if !ok {
			result[hit.Source.Message] = []*KibanaErrorLog{}
		}
		result[hit.Source.Message] = append(result[hit.Source.Message], hit)
	}
	return result
}

func (c *KibanaClient) GetErrorsForMessageKeywords(keywords []string) (*KibanaErrorLogs, error) {
	var shouldClauses []map[string]interface{}
	for _, keyword := range keywords {
		shouldClauses = append(shouldClauses, map[string]interface{}{
			"match_phrase": map[string]interface{}{
				"message": keyword,
			},
		})
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should":               shouldClauses,
				"minimum_should_match": 1,
				"filter": []map[string]interface{}{
					{
						"script": map[string]interface{}{
							"script": map[string]interface{}{
								"source": "doc['correlationId.keyword'].size() > 0 && doc['correlationId.keyword'].value != ''",
								"lang":   "painless",
							},
						},
					},
				},
			},
		},
		"sort": []map[string]interface{}{
			{
				"@timestamp": map[string]interface{}{
					"order": "desc",
				},
			},
		},
	}
	hits, err := c.SearchAll("bms-*", query)
	if err != nil {
		return nil, err
	}
	errorHits := KibanaErrorLogs{}
	for _, hit := range *hits {
		el := KibanaErrorLogSource{}
		if err := mapstructure.Decode(hit.Source, &el); err != nil {
			return nil, fmt.Errorf("failed to convert log source to error source: %e", err)
		}
		errorHits = append(errorHits, &KibanaErrorLog{
			ID:          hit.ID,
			Source:      el,
			Sort:        hit.Sort,
			Coordinates: hit.Coordinates,
		})
	}
	return &errorHits, nil
}

// func (c *KibanaClient) GetTraceForLog(log KibanaLog[KibanaErrorLogSource]) (*KibanaTrace, error) {
// 	// if log is on the outbound route (check against list of known outbound services)
// 	//   get all logs with the same correlation id
// 	//   get one log with a tcr value the same as this correlation id
// 	//   get all logs with the tcr log correlation id
// 	//   sort and deduplicate
// 	// if log is on the inbound route (check against list of known inbound services)
// 	//   get all logs with the same correlation id
// 	//   look for the one log with a tcr value (or tcr value which is uuid)
// 	//   get all logs with the tcr value as correlation id
// 	//   sort and deduplicate
// 	return nil, nil
// }

// func (c *KibanaClient) GetTracesForLogs(logs *KibanaErrorLogs) ([]KibanaTrace, error) {
// 	return []KibanaTrace{}, nil
// }

func (c *KibanaClient) GetErrors() (*KibanaErrorLogs, error) {
	var logs *KibanaErrorLogs
	var err error
	errorsFile, err := os.ReadFile(ErrorsCoordinatesOutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs file: %s", err)
	}
	if err = json.Unmarshal(errorsFile, &logs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal logs: %s", err)
	}
	return logs, nil
}

func (c *KibanaClient) AnalyseErrors() error {
	var logs *KibanaErrorLogs
	var err error
	logsFile, err := os.ReadFile(ErrorsMessageOutputPath)
	if err == nil {
		log.Println("loading logs from local file...")
		if err = json.Unmarshal(logsFile, &logs); err != nil {
			return fmt.Errorf("failed to unmarshal logs file: %s", err)
		}
	} else {
		log.Println("fetching logs from kibana...")
		if logs, err = c.GetErrorsForMessageKeywords(ErrorKeywords); err != nil {
			return fmt.Errorf("failed to get logs: %s", err)
		}
		log.Println("writing logs to local file...")
		if err := output(logs, ErrorsMessageOutputPath); err != nil {
			return fmt.Errorf("failed to write logs: %s", err)
		}
	}

	log.Println("calculating error similarity...")
	comparableLogs := make([]similarity.Comparable, len(*logs))
	for i, l := range *logs {
		comparableLogs[i] = &KibanaLogErrorComparable{l}
	}
	coords, err := similarity.Coordinates(comparableLogs)
	if err != nil {
		return fmt.Errorf("failed to calculate coordinates: %s", err)
	}
	for i, coord := range coords {
		(*logs)[i].Coordinates.Error = coord
	}
	if err := output(logs, ErrorsCoordinatesOutputPath); err != nil {
		return fmt.Errorf("failed to write coordinates: %s", err)
	}

	return nil
}
