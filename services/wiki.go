package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const wikiAPIURL = "https://en.wikipedia.org/api/rest_v1/page/summary/"

type WikiSummary struct {
	Extract string `json:"extract"`
}

func FetchWikiSummary(topic string) (*WikiSummary, error) {
	// URL encode the topic
	encodedTopic := url.QueryEscape(topic)

	// Make request
	resp, err := http.Get(wikiAPIURL + encodedTopic)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wiki API returned status: %d", resp.StatusCode)
	}

	var summary WikiSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, err
	}

	return &summary, nil
}
