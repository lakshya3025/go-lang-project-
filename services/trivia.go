package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
)

const (
	baseURL     = "https://opentdb.com/api.php"
	categoryURL = "https://opentdb.com/api_category.php"
)

type TriviaCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TriviaQuestion struct {
	Category         string   `json:"category"`
	Type             string   `json:"type"`
	Difficulty       string   `json:"difficulty"`
	Question         string   `json:"question"`
	CorrectAnswer    string   `json:"correct_answer"`
	IncorrectAnswers []string `json:"incorrect_answers"`
	// Additional fields for enriched data
	ImageURL       string      `json:"image_url,omitempty"`
	Context        string      `json:"context,omitempty"`
	WordDefinition interface{} `json:"word_definition,omitempty"`
	FunFact        string      `json:"fun_fact,omitempty"`
}

type TriviaResponse struct {
	ResponseCode int              `json:"response_code"`
	Results      []TriviaQuestion `json:"results"`
}

type CategoryResponse struct {
	TriviaCategories []TriviaCategory `json:"trivia_categories"`
}

// FetchCategories retrieves available trivia categories
func FetchCategories() ([]TriviaCategory, error) {
	resp, err := http.Get(categoryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %v", err)
	}
	defer resp.Body.Close()

	var result CategoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode categories: %v", err)
	}

	return result.TriviaCategories, nil
}

// FetchQuizQuestions retrieves questions from the Trivia DB API
func FetchQuizQuestions(categoryID int, difficulty string, amount int) ([]TriviaQuestion, error) {
	// Build URL with query parameters
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Add("amount", fmt.Sprintf("%d", amount))
	if categoryID > 0 {
		q.Add("category", fmt.Sprintf("%d", categoryID))
	}
	if difficulty != "" {
		q.Add("difficulty", strings.ToLower(difficulty))
	}
	u.RawQuery = q.Encode()

	// Make request
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch questions: %v", err)
	}
	defer resp.Body.Close()

	var result TriviaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode questions: %v", err)
	}

	if result.ResponseCode != 0 {
		return nil, fmt.Errorf("API error: response code %d", result.ResponseCode)
	}

	// Clean and decode questions
	for i := range result.Results {
		// Decode base64 if the response is encoded
		if isBase64(result.Results[i].Question) {
			if decoded, err := base64.StdEncoding.DecodeString(result.Results[i].Question); err == nil {
				result.Results[i].Question = string(decoded)
			}
			if decoded, err := base64.StdEncoding.DecodeString(result.Results[i].CorrectAnswer); err == nil {
				result.Results[i].CorrectAnswer = string(decoded)
			}
			for j := range result.Results[i].IncorrectAnswers {
				if decoded, err := base64.StdEncoding.DecodeString(result.Results[i].IncorrectAnswers[j]); err == nil {
					result.Results[i].IncorrectAnswers[j] = string(decoded)
				}
			}
		}

		// Decode HTML entities
		result.Results[i].Question = html.UnescapeString(result.Results[i].Question)
		result.Results[i].CorrectAnswer = html.UnescapeString(result.Results[i].CorrectAnswer)
		for j := range result.Results[i].IncorrectAnswers {
			result.Results[i].IncorrectAnswers[j] = html.UnescapeString(result.Results[i].IncorrectAnswers[j])
		}
	}

	// Create a channel for concurrent enrichment
	enrichChan := make(chan error, amount)

	// Enrich each question concurrently
	for i := range result.Results {
		go func(i int) {
			var err error
			q := &result.Results[i]

			// Use a simple placeholder image instead of Unsplash
			q.ImageURL = fmt.Sprintf("https://placehold.co/600x400/1a1a2e/ffffff/png?text=%s",
				url.QueryEscape(q.Category))

			// Fetch Wikipedia context
			if wiki, err := FetchWikiSummary(q.Category); err == nil {
				q.Context = wiki.Extract
			} else {
				log.Printf("Failed to fetch wiki summary: %v", err)
			}

			enrichChan <- err
		}(i)
	}

	// Check for errors from goroutines
	var enrichErrors []error
	for range result.Results {
		if err := <-enrichChan; err != nil {
			enrichErrors = append(enrichErrors, err)
		}
	}

	if len(enrichErrors) > 0 {
		return nil, fmt.Errorf("encountered %d enrichment errors", len(enrichErrors))
	}

	return result.Results, nil
}

// Helper function to shuffle answers
func ShuffleAnswers(question *TriviaQuestion) []string {
	allAnswers := append([]string{question.CorrectAnswer}, question.IncorrectAnswers...)
	// Fisher-Yates shuffle
	for i := len(allAnswers) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		allAnswers[i], allAnswers[j] = allAnswers[j], allAnswers[i]
	}
	return allAnswers
}

// Helper function to check if a string is base64 encoded
func isBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil && len(s)%4 == 0
}
