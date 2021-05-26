package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	// [START imports]
	language "cloud.google.com/go/language/apiv1"
	//"github.com/go-mail/mail"
	// "github.com/golang/protobuf/proto"
	// "github.com/jdkato/prose/v2"
	"google.golang.org/api/option"
	languagepb "google.golang.org/genproto/googleapis/cloud/language/v1"
	// camundaclientgo "github.com/citilinkru/camunda-client-go"
)

const secret string = "camundaBPMDiscourseTester"

// APIKey is your Discourse API_KEY here
const APIKey = "0476e2a15a1152d8c00c1eeec60bbcec4977e54b87f2d7556a857e814c5a4fb0"

// APIUser is your Discourse User ID
const APIUser = "davidgs"

type lastesPosts struct {
	Users []struct {
		ID             int    `json:"id"`
		Username       string `json:"username"`
		Name           string `json:"name"`
		AvatarTemplate string `json:"avatar_template"`
	} `json:"users"`
	PrimaryGroups []struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		FlairURL     string `json:"flair_url"`
		FlairBgColor string `json:"flair_bg_color"`
		FlairColor   string `json:"flair_color"`
	} `json:"primary_groups"`
	TopicList struct {
		CanCreateTopic bool        `json:"can_create_topic"`
		MoreTopicsURL  string      `json:"more_topics_url"`
		Draft          interface{} `json:"draft"`
		DraftKey       string      `json:"draft_key"`
		DraftSequence  int         `json:"draft_sequence"`
		PerPage        int         `json:"per_page"`
		Topics         []struct {
			ID                 int         `json:"id"`
			Title              string      `json:"title"`
			FancyTitle         string      `json:"fancy_title"`
			Slug               string      `json:"slug"`
			PostsCount         int         `json:"posts_count"`
			ReplyCount         int         `json:"reply_count"`
			HighestPostNumber  int         `json:"highest_post_number"`
			ImageURL           string      `json:"image_url"`
			CreatedAt          time.Time   `json:"created_at"`
			LastPostedAt       time.Time   `json:"last_posted_at"`
			Bumped             bool        `json:"bumped"`
			BumpedAt           time.Time   `json:"bumped_at"`
			Archetype          string      `json:"archetype"`
			Unseen             bool        `json:"unseen"`
			LastReadPostNumber int         `json:"last_read_post_number,omitempty"`
			Unread             int         `json:"unread,omitempty"`
			NewPosts           int         `json:"new_posts,omitempty"`
			Pinned             bool        `json:"pinned"`
			Unpinned           interface{} `json:"unpinned"`
			Visible            bool        `json:"visible"`
			Closed             bool        `json:"closed"`
			Archived           bool        `json:"archived"`
			NotificationLevel  int         `json:"notification_level,omitempty"`
			Bookmarked         bool        `json:"bookmarked"`
			Liked              bool        `json:"liked"`
			Views              int         `json:"views"`
			LikeCount          int         `json:"like_count"`
			HasSummary         bool        `json:"has_summary"`
			LastPosterUsername string      `json:"last_poster_username"`
			CategoryID         int         `json:"category_id"`
			PinnedGlobally     bool        `json:"pinned_globally"`
			FeaturedLink       interface{} `json:"featured_link"`
			HasAcceptedAnswer  bool        `json:"has_accepted_answer"`
			Posters            []struct {
				Extras         interface{} `json:"extras"`
				Description    string      `json:"description"`
				UserID         int         `json:"user_id"`
				PrimaryGroupID interface{} `json:"primary_group_id"`
			} `json:"posters"`
		} `json:"topics"`
	} `json:"topic_list"`
}

// DiscoursePost struct for data from Discourse.
type DiscoursePost struct {
	Post struct {
		ID                int       `json:"id"`
		Name              string    `json:"name"`
		Username          string    `json:"username"`
		AvatarTemplate    string    `json:"avatar_template"`
		CreatedAt         time.Time `json:"created_at"`
		Cooked            string    `json:"cooked"`
		PostNumber        int       `json:"post_number"`
		PostType          int       `json:"post_type"`
		UpdatedAt         time.Time `json:"updated_at"`
		ReplyCount        int       `json:"reply_count"`
		ReplyToPostNumber int       `json:"reply_to_post_number"`
		QuoteCount        int       `json:"quote_count"`
		IncomingLinkCount int       `json:"incoming_link_count"`
		Reads             int       `json:"reads"`
		Score             int       `json:"score"`
		TopicID           int       `json:"topic_id"`
		TopicSlug         string    `json:"topic_slug"`
		TopicTitle        string    `json:"topic_title"`
		CategoryID        int       `json:"category_id"`
		DisplayUsername   string    `json:"display_username"`
		PrimaryGroupName  string    `json:"primary_group_name"`
		Version           int       `json:"version"`
		UserTitle         string    `json:"user_title"`
		ReplyToUser       struct {
			Username       string `json:"username"`
			AvatarTemplate string `json:"avatar_template"`
		} `json:"reply_to_user"`
		Bookmarked                  bool        `json:"bookmarked"`
		Raw                         string      `json:"raw"`
		Moderator                   bool        `json:"moderator"`
		Admin                       bool        `json:"admin"`
		Staff                       bool        `json:"staff"`
		UserID                      int         `json:"user_id"`
		Hidden                      bool        `json:"hidden"`
		TrustLevel                  int         `json:"trust_level"`
		DeletedAt                   interface{} `json:"deleted_at"`
		UserDeleted                 bool        `json:"user_deleted"`
		EditReason                  interface{} `json:"edit_reason"`
		Wiki                        bool        `json:"wiki"`
		ReviewableID                interface{} `json:"reviewable_id"`
		ReviewableScoreCount        int         `json:"reviewable_score_count"`
		ReviewableScorePendingCount int         `json:"reviewable_score_pending_count"`
		TopicPostsCount             int         `json:"topic_posts_count"`
		TopicFilteredPostsCount     int         `json:"topic_filtered_posts_count"`
		TopicArchetype              string      `json:"topic_archetype"`
		CategorySlug                string      `json:"category_slug"`
		Event                       interface{} `json:"event"`
	} `json:"post"`
}

type queryResponse struct {
	Success     bool          `json:"success"`
	Errors      []interface{} `json:"errors"`
	Duration    float64       `json:"duration"`
	ResultCount int           `json:"result_count"`
	Params      struct {
	} `json:"params"`
	Columns      []string `json:"columns"`
	DefaultLimit int      `json:"default_limit"`
	Relations    struct {
	} `json:"relations"`
	Colrender struct {
	} `json:"colrender"`
	Rows [][]string `json:"rows"`
}

// Error represents a handler error. It provides methods for a HTTP status
// code and embeds the built-in error interface.
type Error interface {
	error
	Status() int
}

// StatusError represents an error with an associated HTTP status code.
type StatusError struct {
	Code int
	Err  error
}

// Allows StatusError to satisfy the error interface.
func (se StatusError) Error() string {
	return se.Err.Error()
}

// Status Returns our HTTP status code.
func (se StatusError) Status() int {
	return se.Code
}

// TODO fix comment.
func sendData(s DiscoursePost) {

	topic := s.Post.TopicTitle
	post := s.Post.Raw
	fmt.Println("Event: ", s.Post.Event)
	if topic == "" {
		fmt.Println("We don't care about these.")
		return
	}
	ctx := context.Background()
	client, err := language.NewClient(ctx, option.WithCredentialsFile("credentials.json"))
	if err != nil {
		log.Fatal(err)
		return
	}

	// doc, _ := prose.NewDocument(post)
	// sents := doc.Sentences()
	// fmt.Printf("Letter is %d sentences long.\n", len(sents)) // 2

	// for _, sent := range sents {
	// 	fmt.Printf("Sentence: %s\n", sent.Text)
	sentiment, err := analyzeSentiment(ctx, client, post)
	if err != nil {
		log.Fatal(err)
	}
	if sentiment.DocumentSentiment.Score >= 0 {
		fmt.Printf("Sentiment: %1f, positive\t", sentiment.DocumentSentiment.Score)
	} else {
		fmt.Printf("Sentiment: %1f negative\t", sentiment.DocumentSentiment.Score)
	}
	// }
	// fmt.Println("Post Topic: ", topic)
	// fmt.Println("Post Text: ", post)

}

// [START language_entities_text]

func analyzeEntities(ctx context.Context, client *language.Client, text string) (*languagepb.AnalyzeEntitiesResponse, error) {
	return client.AnalyzeEntities(ctx, &languagepb.AnalyzeEntitiesRequest{
		Document: &languagepb.Document{
			Source: &languagepb.Document_Content{
				Content: text,
			},
			Type: languagepb.Document_PLAIN_TEXT,
		},
		EncodingType: languagepb.EncodingType_UTF8,
	})
}

// [END language_entities_text]

// [START language_sentiment_text]

func analyzeSentiment(ctx context.Context, client *language.Client, text string) (*languagepb.AnalyzeSentimentResponse, error) {
	return client.AnalyzeSentiment(ctx, &languagepb.AnalyzeSentimentRequest{
		Document: &languagepb.Document{
			Source: &languagepb.Document_Content{
				Content: text,
			},
			Type: languagepb.Document_PLAIN_TEXT,
		},
	})
}

// [END language_sentiment_text]

// [START language_syntax_text]

func analyzeSyntax(ctx context.Context, client *language.Client, text string) (*languagepb.AnnotateTextResponse, error) {
	return client.AnnotateText(ctx, &languagepb.AnnotateTextRequest{
		Document: &languagepb.Document{
			Source: &languagepb.Document_Content{
				Content: text,
			},
			Type: languagepb.Document_PLAIN_TEXT,
		},
		Features: &languagepb.AnnotateTextRequest_Features{
			ExtractSyntax: true,
		},
		EncodingType: languagepb.EncodingType_UTF8,
	})
}

// [END language_syntax_text]

// [START language_classify_text]

func classifyText(ctx context.Context, client *language.Client, text string) (*languagepb.ClassifyTextResponse, error) {
	return client.ClassifyText(ctx, &languagepb.ClassifyTextRequest{
		Document: &languagepb.Document{
			Source: &languagepb.Document_Content{
				Content: text,
			},
			Type: languagepb.Document_PLAIN_TEXT,
		},
	})
}

// [END language_classify_text]

// recieve topics from Discourse
func topicEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		fmt.Println("GET Method Not Supported")
		http.Error(w, "GET Method not supported", 400)
	} else {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Got no Data")
		}
		r.Body.Close()
		h := hmac.New(sha256.New, []byte(secret))

		// Write Data to it
		h.Write([]byte(data))

		// Get result and encode as hexadecimal string
		sha := "sha256=" + hex.EncodeToString(h.Sum(nil))
		incomingSha := r.Header.Get("X-Discourse-Event-Signature")
		fmt.Println("Sha256: ", sha)
		fmt.Println("Incoming: ", incomingSha)
		fmt.Println("Raw: ", string(data))
		if sha != incomingSha {
			fmt.Println("Message Header did not pass")
			http.Error(w, "Invalid Message Signature", 500)
		} else {
			var t DiscoursePost
			_ = json.Unmarshal(data, &t)
			var prettyJSON bytes.Buffer
			_ = json.Indent(&prettyJSON, data, "", "\t")
			//fmt.Println(string(prettyJSON.Bytes()))
			sendData(t)
			w.WriteHeader(200)
		}

	}
}

func runQuery(query string) {
	var avg_sent float32 = 0.00
	var high_sent float32 = -100
	var low_sent float32 = 100
	var high_post string = ""
	var low_post string = ""
	sent_num := 0
	fmt.Printf("Running Query: 9\n")
	formValues := url.Values{}
	if len(query) <= 0 {
		query = "connectors"
	}
	formValues.Set("query", query)
	var DefaultClient = &http.Client{}
	urlPlace := "https://forum.camunda.org/admin/plugins/explorer/queries/9" + "/run"
	request, err := http.NewRequest("POST", urlPlace, strings.NewReader(formValues.Encode()))
	if err != nil {
		fmt.Println("Request Object Failure")
		log.Fatal(err)
	}
	request.Header.Set("Content-Type", "multipart/form-data")
	request.Header.Set("Api-Key", APIKey)
	request.Header.Set("Api-Username", APIUser)
	request.Header.Set("Accept", "application/json")
	// var FinalResults []Results
	// var oData = QueryResult{}

	res, err := DefaultClient.Do(request)
	if err != nil {
		fmt.Println("HTTP GET Failed!")
		log.Fatal(err)
	}
	if res.StatusCode != 200 {
		fmt.Println("Got other than Code 200")
		fmt.Println("Code: ", res.StatusCode)
		// log. Fatal(res.StatusCode)
	}
	data, err := ioutil.ReadAll(res.Body)
	// fmt.Println("Data: ", string(data))
	if err != nil {
		fmt.Println("Body Empty")
		log.Fatal("Got no Data")
	}
	res.Body.Close()
	garbage := queryResponse{}
	_ = json.Unmarshal(data, &garbage)
	ctx := context.Background()
	client, err := language.NewClient(ctx, option.WithCredentialsFile("credentials.json"))
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create("./sentiment-output.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	_, err = f.Write([]byte("Post\tSentiment\n"))

	for x := 0; x < len(garbage.Rows); x++ {
		sentiment, err := analyzeSentiment(ctx, client, garbage.Rows[x][0])
		if err != nil {
			log.Fatal(err)
		}

		reg := regexp.MustCompile(`[\n]`) // fix bolds (**foo**)
		translated := string(reg.ReplaceAll([]byte(garbage.Rows[x][0]), []byte("")))
		reg = regexp.MustCompile(`[']`)
		translated = string(reg.ReplaceAll([]byte(translated), []byte("\\'")))
		reg = regexp.MustCompile(`[\t]`)
		translated = string(reg.ReplaceAll([]byte(translated), []byte(" ")))
		// reg = regexp.MustCompile(`[,]`)
		// translated = string(reg.ReplaceAll([]byte(translated), []byte("\\,")))
		avg_sent += sentiment.DocumentSentiment.Score
		sent_num++
		if sentiment.DocumentSentiment.Score >= 0 {
			if sentiment.DocumentSentiment.Score > high_sent{
				high_sent = sentiment.DocumentSentiment.Score
				high_post = translated
			}
			// f.Write([]byte(fmt.Sprintf("'%s'\t", translated)))
			// f.Write([]byte(fmt.Sprintf("%.2f\n", sentiment.DocumentSentiment.Score)))
			// fmt.Printf("'%s'\t%.2f\n", translated, sentiment.DocumentSentiment.Score)
			// fmt.Printf("Sentiment: %1f, positive\n", sentiment.DocumentSentiment.Score)
		} else {
			if sentiment.DocumentSentiment.Score < low_sent {
				low_sent = sentiment.DocumentSentiment.Score
				low_post = translated
			}
			// f.Write([]byte(fmt.Sprintf("'%s'\t", translated)))
			// f.Write([]byte(fmt.Sprintf("%.2f\n", sentiment.DocumentSentiment.Score)))
			// fmt.Printf("'%s'\t%.2f\n", translated, sentiment.DocumentSentiment.Score)
			// fmt.Printf("Sentiment: %1f negative\n", sentiment.DocumentSentiment.Score)
		}
		time.Sleep(1 * time.Second)
	}

	f.Close()
	avg_sent = avg_sent/float32(sent_num)
	fmt.Printf("Average: .2%f\nLow score: .2%f\nLow Post: %s\nHigh score: .2%f\nHigh post: %s\n", avg_sent, low_sent, low_post, high_sent, high_post)
	// var usernameColumn int
	// for z := 0; z < len(oData.Columns); z++ {
	// 	if oData.Columns[z] == "id" || oData.Columns[z] == "user_id" {
	// 		usernameColumn = z
	// 		break
	// 	}
	// }
	// l := len(oData.Rows)
}
func main() {
	fmt.Println("Starting up ... ")
	// fs := http.FileServer(http.Dir("/Users/davidgs/github.com/CamundaHalloween/GoServer/test/"))
	runQuery("connectors")
	// http.HandleFunc("/topic", topicEvent)
	// // http.Handle("/test/", http.StripPrefix("/test", fs)) // set router
	// err := http.ListenAndServeTLS(":9090", "/home/davidgs/.node-red/combined", "/home/davidgs/.node-red/combined", nil) // set listen port
	// // err := http.ListenAndServe(":9090", nil)
	// if err != nil {
	// 	log.Fatal("ListenAndServe: ", err)
	// }
}
