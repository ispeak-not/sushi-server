package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Auth0UserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	UpdatedAt     string `json:"updated_at"`
}

func GetUserInfo(url string, c *gin.Context, token string) (Auth0UserInfo, error) {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Auth0UserInfo{}, fmt.Errorf("http.NewRequest error")
	}
	req.Header.Add("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Auth0UserInfo{}, fmt.Errorf("http.DefaultClient.Do error")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Auth0UserInfo{}, fmt.Errorf("io.ReadAll error")
	}

	userInfo := Auth0UserInfo{}
	err = json.Unmarshal(body, &userInfo)
	if err != nil {
		return Auth0UserInfo{}, fmt.Errorf("json.Unmarshal error")
	}

	return userInfo, nil
}
