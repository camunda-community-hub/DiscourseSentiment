package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type APIQuery struct {
	URL string `json:"url"`
	QUERY string `json:"query"`
	APIKEY string `json:"apikey"`
	USERNAME string `json:"username"`
	APIResponse APIResponse `json:"apiresponse"`
}

type APIResponse struct {
	Posts               []Post `json:"posts"`
	Users               []interface{} `json:"users"`
	Categories          []interface{} `json:"categories"`
	Groups              []interface{} `json:"groups"`
	GroupedSearchResult struct {
		MorePosts           interface{}   `json:"more_posts"`
		MoreUsers           interface{}   `json:"more_users"`
		MoreCategories      interface{}   `json:"more_categories"`
		Term                string        `json:"term"`
		SearchLogID         int           `json:"search_log_id"`
		MoreFullPageResults interface{}   `json:"more_full_page_results"`
		CanCreateTopic      bool          `json:"can_create_topic"`
		Error               interface{}   `json:"error"`
		PostIds             []interface{} `json:"post_ids"`
		UserIds             []interface{} `json:"user_ids"`
		CategoryIds         []interface{} `json:"category_ids"`
		GroupIds            []interface{} `json:"group_ids"`
	} `json:"grouped_search_result"`
}

type Post struct {
	AvatarTemplate string    `json:"avatar_template"`
	Blurb          string    `json:"blurb"`
	CreatedAt      time.Time `json:"created_at"`
	ID             int       `json:"id"`
	LikeCount      int       `json:"like_count"`
	Name           string    `json:"name"`
	PostNumber     int       `json:"post_number"`
	TopicID        int       `json:"topic_id"`
	Username       string    `json:"username"`
}

func NewAPIQuery(url string, query string, apikey string, username string) *APIQuery {
	return &APIQuery{
		URL: url,
		QUERY: query,
		APIKEY: apikey,
		USERNAME: username,
		APIResponse: APIResponse{},
	}
}

func (a *APIQuery) SendRequest() error {
	// Request (GET https://forum.camunda.io/search.json?q=%22human%20workflow%22%20max_posts:1000)

	// Create client
	client := &http.Client{}

	// Create request
	req, err := http.NewRequest("GET", a.URL + a.QUERY + "%20max_posts:1500", nil)
	if err != nil {
		return err
	}
	fmt.Println("Request:", req.URL.String())
	// Headers
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Api-Username", a.USERNAME)
	req.Header.Add("Api-Key", a.APIKEY)

	//"0476e2a15a1152d8c00c1eeec60bbcec4977e54b87f2d7556a857e814c5a4fb0")

	parseFormErr := req.ParseForm()
	if parseFormErr != nil {
		fmt.Println(parseFormErr)
		return parseFormErr
	}

	// Fetch Request
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Failure : ", err)
		return err
	}

	// Read Response Body
	respBody, _ := ioutil.ReadAll(resp.Body)

	// Display Results
	fmt.Println("response Status : ", resp.Status)
	fmt.Println("response Headers : ", resp.Header)
	// fmt.Println("response Body : ", string(respBody))
	ar := APIResponse{}
	err = json.Unmarshal(respBody, &ar)
	if err != nil {
		fmt.Println(err)
	}
	a.APIResponse = ar
	fmt.Println(len(ar.Posts))
	return nil
}

// func main() {
// 	aq := NewAPIQuery("https://forum.camunda.org/search.json?q=", "human%20workflow", "0476e2a15a1152d8c00c1eeec60bbcec4977e54b87f2d7556a857e814c5a4fb0", "davidgs")
// 	err := aq.sendRequest()
// 		if err != nil {
// 		fmt.Println(err)
// 	}
// }
