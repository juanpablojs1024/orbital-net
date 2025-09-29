package httpclient

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/hashicorp/consul/api"
)

var (
	simulatorBaseURL string
	once             sync.Once
)

func getSimulatorBaseURL() (string, error) {
	var err error
	once.Do(func() {
		client, e := api.NewClient(api.DefaultConfig())
		if e != nil {
			err = e
			return
		}

		services, _, e := client.Health().Service("simulator", "", true, nil)
		if e != nil {
			err = e
			return
		}

		if len(services) == 0 {
			err = fmt.Errorf("no simulator services found in Consul")
			return
		}

		svc := services[0].Service
		simulatorBaseURL = fmt.Sprintf("http://%s:%d", svc.Address, svc.Port)
	})

	if err != nil {
		return "", err
	}
	return simulatorBaseURL, nil
}

func fetchJSON(url string) (interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed interface{}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func FetchVisibility() []interface{} {
	baseURL, err := getSimulatorBaseURL()
	if err != nil {
		log.Fatalf("❌ Failed to discover simulator service: %v", err)
	}

	visibility, err := fetchJSON(fmt.Sprintf("%s/visibility", baseURL))
	if err != nil {
		log.Fatalf("❌ Failed to fetch visibility: %v", err)
	}

	matrix, ok := visibility.([]interface{})
	if !ok {
		log.Fatal("❌ Failed to cast visibility to []interface{}")
	}
	return matrix
}

func FetchNodes() []interface{} {
	baseURL, err := getSimulatorBaseURL()
	if err != nil {
		log.Fatalf("❌ Failed to discover simulator service: %v", err)
	}

	data, err := fetchJSON(fmt.Sprintf("%s/positions", baseURL))
	if err != nil {
		log.Fatalf("❌ Failed to fetch positions: %v", err)
	}

	items, ok := data.([]interface{})
	if !ok {
		log.Fatal("❌ expected JSON array at top level")
	}
	return items
}
