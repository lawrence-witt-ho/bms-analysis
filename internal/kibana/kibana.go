package kibana

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/atoscerebro/bms-analysis/internal/config"
	"github.com/atoscerebro/bms-analysis/internal/similarity"
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

var ErrorMessageOutputPath = "test-error-message-output.json"
var ErrorCoordinatesOutputPath = "test-error-coordinates-output.json"

type KibanaLogSource struct {
	CorrelationId string    `json:"correlationId"`
	TCR           string    `json:"tcr"`
	Environment   string    `json:"environment"`
	HttpStatus    int32     `json:"httpStatus"`
	Message       string    `json:"message"`
	Microservice  string    `json:"microservice"`
	ErrorMessage  string    `json:"errorMessage"`
	TimeStamp     time.Time `json:"@timestamp"`
}

type KibanaLogCoordinates struct {
	Error similarity.Coordinate `json:"error"`
}

type KibanaLog struct {
	ID          string               `json:"_id"`
	Source      KibanaLogSource      `json:"_source"`
	Sort        []interface{}        `json:"sort"`
	Coordinates KibanaLogCoordinates `json:"coordinates"`
}

type KibanaLogErrorComparable struct {
	*KibanaLog
}

func (kl *KibanaLogErrorComparable) Metric() string {
	return kl.Source.ErrorMessage
}

type KibanaLogs []*KibanaLog

func (kl *KibanaLogs) ByMessage() map[string][]*KibanaLog {
	result := map[string][]*KibanaLog{}
	for _, hit := range *kl {
		_, ok := result[hit.Source.Message]
		if !ok {
			result[hit.Source.Message] = []*KibanaLog{}
		}
		result[hit.Source.Message] = append(result[hit.Source.Message], hit)
	}
	return result
}

type KibanaHits struct {
	Hits  KibanaLogs `json:"hits"`
	Total float64    `json:"total"`
}

type KibanaSearchResult struct {
	Hits KibanaHits `json:"hits"`
}

type KibanaTrace struct {
	Logs KibanaLogs
}

type KibanaClient struct {
	URL      string
	Username string
	Password string
}

func NewKibanaClient(cfg *config.Config) *KibanaClient {
	return &KibanaClient{
		URL:      cfg.KibanaURL,
		Username: cfg.LDAPUsername,
		Password: cfg.LDAPPassword,
	}
}

func output(o interface{}, path string) error {
	outputBytes, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %s", err)
	}
	err = os.WriteFile(path, outputBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output: %s", err)
	}
	return nil
}

func (c *KibanaClient) Search(query map[string]interface{}) (*KibanaSearchResult, error) {
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %s", err)
	}
	url := fmt.Sprintf("%s/%s", c.URL, "elasticsearch/bms-*/_search")

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(queryBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to created request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("kbn-version", "6.8.21")
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.Username, c.Password)))
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", auth))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %s", err)
	}
	defer res.Body.Close()

	output := KibanaSearchResult{}
	err = json.NewDecoder(res.Body).Decode(&output)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response body: %s", err)
	}

	return &output, nil
}

func (c *KibanaClient) SearchAll(query map[string]interface{}) (*KibanaLogs, error) {
	size := 1000
	var search_after []interface{}
	hits := KibanaLogs{}
	for {
		subQuery := maps.Clone(query)
		subQuery["size"] = size
		if len(search_after) != 0 {
			subQuery["search_after"] = search_after
		}
		log.Printf("querying with size '%d' and search_after '%v'...", size, search_after)
		subOutput, err := c.Search(subQuery)
		if err != nil {
			return nil, err
		}
		totalLength := subOutput.Hits.Total
		pageLength := len(subOutput.Hits.Hits)
		log.Printf("returned '%d' hits of '%s' total", pageLength, strconv.FormatFloat(totalLength, 'f', -1, 64))
		hits = append(hits, subOutput.Hits.Hits...)
		if pageLength < size {
			break
		}
		search_after = subOutput.Hits.Hits[pageLength-1].Sort
	}
	return &hits, nil
}

func (c *KibanaClient) GetLogsForMessageKeywords(keywords []string) (*KibanaLogs, error) {
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
	hits, err := c.SearchAll(query)
	if err != nil {
		return nil, err
	}

	return hits, nil
}

func (c *KibanaClient) GetTraceForLog(log KibanaLog) (*KibanaTrace, error) {
	// if log is on the outbound route (check against list of known outbound services)
	//   get all logs with the same correlation id
	//   get one log with a tcr value the same as this correlation id
	//   get all logs with the tcr log correlation id
	//   sort and deduplicate
	// if log is on the inbound route (check against list of known inbound services)
	//   get all logs with the same correlation id
	//   look for the one log with a tcr value (or tcr value which is uuid)
	//   get all logs with the tcr value as correlation id
	//   sort and deduplicate
	return nil, nil
}

func (c *KibanaClient) GetTracesForLogs(logs *KibanaLogs) ([]KibanaTrace, error) {
	return []KibanaTrace{}, nil
}

func (c *KibanaClient) Logs() (*KibanaLogs, error) {
	var logs *KibanaLogs
	var err error
	errorsFile, err := os.ReadFile(ErrorCoordinatesOutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs file: %s", err)
	}
	if err = json.Unmarshal(errorsFile, &logs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal logs: %s", err)
	}
	return logs, nil
}

func (c *KibanaClient) Analyse() error {
	var logs *KibanaLogs
	var err error
	logsFile, err := os.ReadFile(ErrorMessageOutputPath)
	if err == nil {
		log.Println("loading logs from local file...")
		if err = json.Unmarshal(logsFile, &logs); err != nil {
			return fmt.Errorf("failed to unmarshal logs file: %s", err)
		}
	} else {
		log.Println("fetching logs from kibana...")
		if logs, err = c.GetLogsForMessageKeywords(ErrorKeywords); err != nil {
			return fmt.Errorf("failed to get logs: %s", err)
		}
		log.Println("writing logs to local file...")
		if err := output(logs, ErrorMessageOutputPath); err != nil {
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
	if err := output(logs, ErrorCoordinatesOutputPath); err != nil {
		return fmt.Errorf("failed to write coordinates: %s", err)
	}

	return nil
}
