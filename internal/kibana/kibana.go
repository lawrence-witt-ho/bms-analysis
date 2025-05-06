package kibana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/atoscerebro/bms-analysis/internal/config"
	"github.com/elastic/go-elasticsearch/v9"
)

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

func (c *KibanaClient) Search() error {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			c.URL,
		},
		Username: c.Username,
		Password: c.Password,
	})
	if err != nil {
		return err
	}

	i, err := es.Info()
	if err != nil {
		return err
	}
	defer i.Body.Close()
	log.Println(i)

	// query := map[string]interface{}{
	// 	"query": map[string]interface{}{
	// 		"bool": map[string]interface{}{
	// 			"must": []interface{}{
	// 				map[string]interface{}{
	// 					"match_all": map[string]interface{}{},
	// 				},
	// 			},
	// 			"filter": []interface{}{
	// 				map[string]interface{}{
	// 					"range": map[string]interface{}{
	// 						"@timestamp": map[string]string{
	// 							"gte": "now-15m",
	// 							"lte": "now",
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 	},
	// 	"sort": []interface{}{
	// 		map[string]interface{}{
	// 			"@timestamp": map[string]string{
	// 				"order": "desc",
	// 			},
	// 		},
	// 	},
	// 	"size": 50,
	// }
	query := map[string]interface{}{
		// "query": map[string]interface{}{
		// 	"range": map[string]interface{}{
		// 		"@timestamp": map[string]interface{}{
		// 			"gte": "now-15m/m",
		// 			"lte": "now",
		// 		},
		// 	},
		// },
		// "sort": []map[string]interface{}{
		// 	{
		// 		"@timestamp": map[string]interface{}{
		// 			"order": "desc",
		// 		},
		// 	},
		// },
		"size": 5,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return fmt.Errorf("error encoding query: %s", err)
	}
	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("bms-*"),
		es.Search.WithBody(&buf),
		es.Search.WithTrackTotalHits(true),
		es.Search.WithPretty(),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return fmt.Errorf("error parsing the response body: %s", err)
	}
	by, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile("output.json", by, 0644)
	return err
}
