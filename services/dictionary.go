package services

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const dictionaryAPIURL = "https://api.dictionaryapi.dev/api/v2/entries/en/"

type DictionaryResponse struct {
	Word     string `json:"word"`
	Phonetic string `json:"phonetic"`
	Meanings []struct {
		PartOfSpeech string `json:"partOfSpeech"`
		Definitions  []struct {
			Definition string `json:"definition"`
			Example    string `json:"example,omitempty"`
		} `json:"definitions"`
	} `json:"meanings"`
}

func FetchWordDefinition(word string) (interface{}, error) {
	// Make request
	resp, err := http.Get(dictionaryAPIURL + word)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dictionary API returned status: %d", resp.StatusCode)
	}

	var definitions []DictionaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&definitions); err != nil {
		return nil, err
	}

	if len(definitions) == 0 {
		return nil, fmt.Errorf("no definitions found")
	}

	return definitions[0], nil
}
