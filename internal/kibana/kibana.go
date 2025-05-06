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
)

// Get all logs which have the message property in our list
// Get the correlation or tcr id from each log (prefer tcr?)
// Search for all logs which have that correlation or tcr id, time ordered
// Calculate the path taken (using an existing service map?)
// Output a common data format with the path, and normalised error messages so we can cross compare

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

type KibanaLogSource struct {
	CorrelationId string    `json:"correlationId"`
	TCR           string    `json:"tcr"`
	Environment   string    `json:"environment"`
	Message       string    `json:"message"`
	Microservice  string    `json:"microservice"`
	ErrorMessage  string    `json:"errorMessage"`
	TimeStamp     time.Time `json:"@timestamp"`
}
type KibanaLog struct {
	ID     string          `json:"_id"`
	Source KibanaLogSource `json:"_source"`
	Sort   []interface{}   `json:"sort"`
}

type KibanaHits struct {
	Hits  []KibanaLog `json:"hits"`
	Total float64     `json:"total"`
}

type KibanaSearchResult struct {
	Hits KibanaHits `json:"hits"`
}

type KibanaTrace struct {
	Logs []KibanaLog
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

func (c *KibanaClient) SearchAll(query map[string]interface{}) ([]KibanaLog, error) {
	size := 1000
	var search_after []interface{}
	hits := []KibanaLog{}
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
	return hits, nil
}

func (c *KibanaClient) GetLogsForMessageKeywords(keywords []string) (map[string][]KibanaLog, error) {
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
	result := map[string][]KibanaLog{}
	for _, hit := range hits {
		_, ok := result[hit.Source.Message]
		if !ok {
			result[hit.Source.Message] = []KibanaLog{}
		}
		result[hit.Source.Message] = append(result[hit.Source.Message], hit)
	}

	return result, nil
}

func (c *KibanaClient) GetTraceForLog(log KibanaLog) (*KibanaTrace, error) {
	return nil, nil
}

func (c *KibanaClient) GetTracesForLogs(logs []KibanaLog) ([]KibanaTrace, error) {
	return []KibanaTrace{}, nil
}

func (c *KibanaClient) AnalyseErrorKeywords() error {
	o, err := c.GetLogsForMessageKeywords(ErrorKeywords)
	if err != nil {
		return err
	}
	return output(o, ErrorMessageOutputPath)
}
