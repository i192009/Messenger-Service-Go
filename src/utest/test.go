package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func main() {
	suiteKey := "suite1ludm7uriqvcrw87"
	suiteSecret := "fuCrRczSgkLV7U3zNo4gEBvjl__vD-e8J5wP8yqUepT0SbZ1wk8dmf0Bls6MLVsc"
	suiteTicket := "TneeqHGoVQW9Bdn1qINRLJT4GOVANQb3AVWM3K9bO4oRDZa82kKkAWpCkSGQkp4Mk8EoQ8YssS1X5xUr4dblL3"
	authCorpId := "ding79c1b205f2f9f172ffe93478753d9884"
	// Create Signature String
	timestamp := time.Now().UnixNano() / 1e6
	signatureString := fmt.Sprintf("%d\n%s", timestamp, suiteTicket)
	// Create Signature
	h := hmac.New(sha256.New, []byte(suiteSecret))
	h.Write([]byte(signatureString))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	query := url.Values{}
	query.Add("accessKey", suiteKey)
	query.Add("timestamp", strconv.FormatInt(timestamp, 10))
	query.Add("suiteTicket", suiteTicket)
	query.Add("signature", signature)

	// Get DingTalk AppAccessToken
	reqUrl := url.URL{
		Scheme:   "https",
		Host:     "oapi.dingtalk.com",
		Path:     "service/get_auth_info",
		RawQuery: query.Encode(),
	}

	// Request body
	body, _ := json.Marshal(
		gin.H{
			"auth_corpid": authCorpId,
			"suite_key":   suiteKey,
		})
	requestBody := bytes.NewBuffer(body)

	// Create Request
	req, err := http.NewRequest("POST", reqUrl.String(), requestBody)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Send Request
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
	}(resp.Body)

	//Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Unmarshal the response body
	var data map[string]interface{}
	err = json.Unmarshal(respBody, &data)

	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(data)
}
