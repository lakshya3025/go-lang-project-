package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"quizapp/models"
)

const githubAuthURL = "https://github.com/login/oauth/authorize"
const githubTokenURL = "https://github.com/login/oauth/access_token"
const githubUserURL = "https://api.github.com/user"

func GetGithubAuthURL() string {
	return fmt.Sprintf("%s?client_id=%s&scope=user:email",
		githubAuthURL,
		os.Getenv("GITHUB_CLIENT_ID"),
	)
}

func GetGithubUser(code string) (*models.GithubUser, error) {
	// Exchange code for access token
	tokenReq, _ := http.NewRequest("POST", githubTokenURL, nil)
	q := tokenReq.URL.Query()
	q.Add("client_id", os.Getenv("GITHUB_CLIENT_ID"))
	q.Add("client_secret", os.Getenv("GITHUB_CLIENT_SECRET"))
	q.Add("code", code)
	tokenReq.URL.RawQuery = q.Encode()
	tokenReq.Header.Add("Accept", "application/json")

	tokenResp, err := http.DefaultClient.Do(tokenReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %v", err)
	}
	defer tokenResp.Body.Close()

	var tokenData struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %v", err)
	}

	// Get user data
	userReq, _ := http.NewRequest("GET", githubUserURL, nil)
	userReq.Header.Add("Authorization", "token "+tokenData.AccessToken)
	userReq.Header.Add("Accept", "application/json")

	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get user data: %v", err)
	}
	defer userResp.Body.Close()

	var user models.GithubUser
	if err := json.NewDecoder(userResp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user data: %v", err)
	}

	return &user, nil
}
