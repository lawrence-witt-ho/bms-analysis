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

	"github.com/atoscerebro/bms-analysis/internal/config"
	"github.com/atoscerebro/bms-analysis/internal/similarity"
)

type KibanaLogCoordinates struct {
	Error similarity.Coordinate `json:"error"`
}

type KibanaLog struct {
	ID          string                 `json:"_id"`
	Source      map[string]interface{} `json:"_source"`
	Sort        []interface{}          `json:"sort"`
	Coordinates KibanaLogCoordinates   `json:"coordinates"`
}

type KibanaLogs []*KibanaLog

type KibanaHits struct {
	Hits  KibanaLogs `json:"hits"`
	Total float64    `json:"total"`
}

type KibanaSearchResult struct {
	Hits KibanaHits `json:"hits"`
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

func (c *KibanaClient) Search(filter string, query map[string]interface{}) (*KibanaSearchResult, error) {
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %s", err)
	}
	url := fmt.Sprintf("%s/%s", c.URL, fmt.Sprintf("elasticsearch/%s/_search", filter))

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

func (c *KibanaClient) SearchAll(filter string, query map[string]interface{}) (*KibanaLogs, error) {
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
		subOutput, err := c.Search(filter, subQuery)
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
