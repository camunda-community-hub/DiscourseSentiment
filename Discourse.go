package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	// [START imports]
	language "cloud.google.com/go/language/apiv1"
	camundaclientgo "github.com/citilinkru/camunda-client-go/v2"
	"github.com/citilinkru/camunda-client-go/v2/processor"
	"github.com/go-gomail/gomail"
	"github.com/matcornic/hermes/v2"
	"google.golang.org/api/option"
	languagepb "google.golang.org/genproto/googleapis/cloud/language/v1"
)
const serverBase = "https://sentiment.camunda.com"
const secret string = "camundaBPMDiscourseTester"
const camundaServer = serverBase + ":8443/engine-rest"

// APIKey is your Discourse API_KEY here
const PlatformAPIKey = "0476e2a15a1152d8c00c1eeec60bbcec4977e54b87f2d7556a857e814c5a4fb0"
const PlatformURL = "https://forum.camunda.org/admin/plugins/explorer/queries/9/run"
const BPMNAPIKey = "d909293c78ee5cc6fa055419da5f9682a595341a2eb1acb02ce17d7bcb7c060c"
const BPMNUrl = "https://forum.bpmn.io/admin/plugins/explorer/queries/2/run"

const CloudAPIKey = "b43ac8b6f5b178860d20e84ddca232a7c8fea3cb1a195ea03b16ab6b92f2b6d4"
const CloudURL = "https://forum.camunda.io/admin/plugins/explorer/queries/3/run"

// APIUser is your Discourse User ID
const APIUser = "davidgs"

type Submitter struct {
	Username   string `json:"username"`
	Email      string `json:"emailAddress"`
	Forum      string `json:"community"`
	SearchTerm string `json:"searchterm"`
}

var submitter = Submitter{}

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

type Relations struct {
	Post []struct {
		ID             int    `json:"id"`
		TopicID        int    `json:"topic_id"`
		PostNumber     int    `json:"post_number"`
		Excerpt        string `json:"excerpt"`
		Username       string `json:"username"`
		AvatarTemplate string `json:"avatar_template"`
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
		Post []struct {
			ID             int    `json:"id"`
			TopicID        int    `json:"topic_id"`
			PostNumber     int    `json:"post_number"`
			Excerpt        string `json:"excerpt"`
			Username       string `json:"username"`
			AvatarTemplate string `json:"avatar_template"`
		} `json:"post"`
	} `json:"relations"`
	Colrender struct {
		Num0 string `json:"0"`
	} `json:"colrender"`
	Rows [][]string `json:"rows"`
}

type CamundaProcess []struct {
	ID                       string      `json:"id"`
	BusinessKey              interface{} `json:"businessKey"`
	ProcessDefinitionID      string      `json:"processDefinitionId"`
	ProcessDefinitionKey     string      `json:"processDefinitionKey"`
	ProcessDefinitionName    string      `json:"processDefinitionName"`
	ProcessDefinitionVersion int         `json:"processDefinitionVersion"`
	StartTime                string      `json:"startTime"`
	EndTime                  interface{} `json:"endTime"`
	RemovalTime              interface{} `json:"removalTime"`
	DurationInMillis         interface{} `json:"durationInMillis"`
	StartUserID              interface{} `json:"startUserId"`
	StartActivityID          string      `json:"startActivityId"`
	DeleteReason             interface{} `json:"deleteReason"`
	RootProcessInstanceID    string      `json:"rootProcessInstanceId"`
	SuperProcessInstanceID   interface{} `json:"superProcessInstanceId"`
	SuperCaseInstanceID      interface{} `json:"superCaseInstanceId"`
	CaseInstanceID           interface{} `json:"caseInstanceId"`
	TenantID                 interface{} `json:"tenantId"`
	State                    string      `json:"state"`
}

type ProcessQueryResult []struct {
	Links          []interface{} `json:"links"`
	ID             string        `json:"id"`
	DefinitionID   string        `json:"definitionId"`
	BusinessKey    string        `json:"businessKey"`
	CaseInstanceID string        `json:"caseInstanceId"`
	Ended          bool          `json:"ended"`
	Suspended      bool          `json:"suspended"`
	TenantID       interface{}   `json:"tenantId"`
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
	sentiment, err := analyzeSentiment(ctx, client, post)
	if err != nil {
		log.Fatal(err)
	}
	if sentiment.DocumentSentiment.Score >= 0 {
		fmt.Printf("Sentiment: %1f, positive\t", sentiment.DocumentSentiment.Score)
	} else {
		fmt.Printf("Sentiment: %1f negative\t", sentiment.DocumentSentiment.Score)
	}
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
		http.Error(w, "GET Method not supported", 400)
	} else {
		data, err := ioutil.ReadAll(r.Body)
		// fmt.Println(string(data))
		if err != nil {
			log.Fatal("Got no Data")
		}
		r.Body.Close()
		qString := html.UnescapeString(string(data))
		fmt.Println(qString)
		formFields := strings.Split(qString, "&")
		for x := 0; x < len(formFields); x++ {
			fmt.Println(formFields[x])
			if strings.HasPrefix(formFields[x], "username") {
				submitter.Username = strings.ReplaceAll(strings.Split(formFields[x], "=")[1], "+", " ")
			} else if strings.HasPrefix(formFields[x], "emailAddress") {
				submitter.Email = strings.ReplaceAll(strings.Split(formFields[x], "=")[1], "%40", "@")
			} else if strings.HasPrefix(formFields[x], "community") {
				submitter.Forum = strings.Split(formFields[x], "=")[1]
			} else if strings.HasPrefix(formFields[x], "searchterm") {
				submitter.SearchTerm = strings.ReplaceAll(strings.Split(formFields[x], "=")[1], "+", " ")
			} else {

			}
		}
		// We only allow camunda employees, with their Camunda email address.
		if !strings.HasSuffix(submitter.Email, "camunda.com") {
			http.Redirect(w, r, "https://sentiment.camunda.com/sorry.html", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "https://sentiment.camunda.com/thanks.html", http.StatusSeeOther)
		var variables = map[string]camundaclientgo.Variable{}
		variables["username"] = camundaclientgo.Variable{
			Value: submitter.Username,
			Type:  "string",
		}
		variables["email"] = camundaclientgo.Variable{
			Value: submitter.Email,
			Type:  "string",
		}
		variables["forum"] = camundaclientgo.Variable{
			Value: submitter.Forum,
			Type:  "string",
		}
		variables["searchterm"] = camundaclientgo.Variable{
			Value: submitter.SearchTerm,
			Type:  "string",
		}
		variables["queue"] = camundaclientgo.Variable{
			Value: "true",
			Type:  "boolean",
		}
		businessKey := fmt.Sprintf("SentimentAnalysis%s", RandStringBytesRmndr(8))
		opts := camundaclientgo.ClientOptions{}
		opts.EndpointUrl = camundaServer
		opts.Timeout = time.Second * 20
		client := camundaclientgo.NewClient(opts)
		reqMessage := camundaclientgo.ReqMessage{}
		reqMessage.BusinessKey = businessKey
		reqMessage.ProcessVariables = &variables
		reqMessage.MessageName = "submit_form"
		err = client.Message.SendMessage(&reqMessage)
		if err != nil {
			log.Printf("Error starting process: %s\n", err)
			return
		}
	}
}

func sendEmail(avg_sent float32, low_sent float32, low_post string, lowURL string, high_sent float32, high_post string, highURL string) error {

	h := hermes.Hermes{
		Theme: new(SentimentTheme),
		Product: hermes.Product{
			// Appears in header & footer of e-mails
			Name: "Camunda, Inc.",
			Link: "https://sentiment.camunda.com/",
			// Optional product logo
			Logo: "https://sentiment.camunda.com:443/images/Logo_White.svg",
			Copyright: "© 2021 Camunda, Inc.",
		},
	}
	email := hermes.Email{
		Body: hermes.Body{
			Name: submitter.Username,
			Intros: []string{
				fmt.Sprintf("Here's what we found in analysing the term '%s'", submitter.SearchTerm),
				fmt.Sprintf("The overall average sentiment was: %.2f", avg_sent),
				fmt.Sprintf("The high sentiment was: %.2f", high_sent),
				fmt.Sprintf("Here's an excerpt from that post: \n<blockquote>%s</blockquote>\n", high_post),
				fmt.Sprintf("You can read the whole post <a href=\"%s\">here</a>", highURL),
				fmt.Sprintf("The low sentiment was: %.2f", low_sent),
				fmt.Sprintf("Here's an excerpt from that post: \n<blockquote>%s</blockquote>\n", low_post),
				fmt.Sprintf("You can read the whole post <a href=\"%s\">here</a>", lowURL),
			},
			Dictionary: []hermes.Entry {
				{Key: "Submitter Name", Value: submitter.Username},
				{Key: "Submitter Email", Value: submitter.Email},
				{Key: "Camunda Forum Searched", Value: submitter.Forum},
				{Key: "Search Term", Value: submitter.SearchTerm},
			},
		},
	}

	// Generate an HTML email with the provided contents (for modern clients)
	emailBody, err := h.GenerateHTML(email)
	if err != nil {
		return err
	}

	// Generate the plaintext version of the e-mail (for clients that do not support xHTML)
	emailText, err := h.GeneratePlainText(email)
	if err != nil {
		return err
	}

	// Optionally, preview the generated HTML e-mail by writing it to a local file
	sendEmails := os.Getenv("HERMES_SEND_EMAILS") == "true"
	if sendEmails {
		port, _ := strconv.Atoi(os.Getenv("HERMES_SMTP_PORT"))
		password := os.Getenv("HERMES_SMTP_PASSWORD")
		SMTPUser := os.Getenv("HERMES_SMTP_USER")
		smtpConfig := smtpAuthentication{
			Server:         os.Getenv("HERMES_SMTP_SERVER"),
			Port:           port,
			SenderEmail:    os.Getenv("HERMES_SENDER_EMAIL"),
			SenderIdentity: os.Getenv("HERMES_SENDER_IDENTITY"),
			SMTPPassword:   password,
			SMTPUser:       SMTPUser,
		}
		options := sendOptions{
			To: submitter.Email,
		}
		options.Subject = "Camunda Sentiment Analysis Results"
		fmt.Printf("Sending email '%s'...\n", options.Subject)
		err = send(smtpConfig, options, string(emailBody), string(emailText))
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	client := camundaclientgo.NewClient(camundaclientgo.ClientOptions{EndpointUrl: serverBase +":8443/engine-rest",
		Timeout: time.Second * 20,
	})
	logger := func(err error) {
		fmt.Println(err.Error())
	}
	asyncResponseTimeout := 5000
	proc := processor.NewProcessor(client, &processor.ProcessorOptions{
		WorkerId:                  "SentimentAnalyzer",
		LockDuration:              time.Second * 240,
		MaxTasks:                  10,
		MaxParallelTaskPerHandler: 100,
		LongPollingTimeout:        20 * time.Second,
		AsyncResponseTimeout:      &asyncResponseTimeout,
	}, logger)
	fmt.Println("Sentiment Processor started ... ")
	proc.AddHandler(
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "checkQueue"},
		},
		func(ctx *processor.Context) error {
			return queueStatus(ctx.Task.Variables, ctx)
		},
	)
	proc.AddHandler(
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "runThisAnalysis"},
		},
		func(ctx *processor.Context) error {
			return runAnalysis(ctx.Task.Variables, ctx)
		},
	)
	proc.AddHandler(
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "checkForNext"},
		},
		func(ctx *processor.Context) error {
			return getNext(ctx.Task.Variables, ctx)
		},
	)
	proc.AddHandler(
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "fireNext"},
		},
		func(ctx *processor.Context) error {
			return fireNext(ctx.Task.Variables, ctx)
		},
	)
	fmt.Println("Starting up ... ")
	http.HandleFunc("/sentiment", topicEvent)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	// // http.Handle("/test/", http.StripPrefix("/test", fs)) // set router
	err := http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/sentiment.camunda.com/cert.pem", "/etc/letsencrypt/live/sentiment.camunda.com/privkey.pem", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func getUrl(url string) ProcessQueryResult {
	var DefaultClient = &http.Client{}
	fmt.Println(url)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Request Object Failure")
		log.Fatal(err)
	}

	res, err := DefaultClient.Do(request)
	if err != nil {
		fmt.Println("HTTP GET Failed!", url)
		log.Fatal(err)
	}
	if res.StatusCode >= 220 {
		fmt.Println("Got other than Code 200/204")
		fmt.Println("Code: ", res.StatusCode)
		// log. Fatal(res.StatusCode)
	}
	data, err := ioutil.ReadAll(res.Body)
	// fmt.Println("Data: ", string(data))
	if err != nil {
		log.Fatal("Got no Data")
	}
	res.Body.Close()
	garbage := ProcessQueryResult{}
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, data, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(prettyJSON.Bytes()))
	err = json.Unmarshal(data, &garbage)
	if err != nil {
		log.Fatal(err)
	}
	return garbage
}

func fireNext(newVars map[string]camundaclientgo.Variable, contx *processor.Context) error {
	fmt.Println("Firing next process ...")
	// complete the process first, else it shows up as still active and all hell breaks loose
	err := contx.Complete(processor.QueryComplete{Variables: &contx.Task.Variables})
	if err != nil {
		log.Fatal(err)
	}
	// get a list of waiting processes
	garbage := getUrl(camundaServer + "/process-instance?businessKeyLike=SentimentAnalysis&processDefinitionKey=SentimentHandler&variable=queue_eg_true")
	if garbage == nil {
		log.Fatal("Failed to get next for unknown reasons")
	}
	if len(garbage) >= 1 { // there's at least one process waiting

		opts := camundaclientgo.ClientOptions{}
		opts.EndpointUrl = camundaServer
		opts.Timeout = time.Second * 20
		client := camundaclientgo.NewClient(opts)
		reqMessage := camundaclientgo.ReqMessage{}
		reqMessage.BusinessKey = garbage[0].BusinessKey
		reqMessage.ProcessVariables = &contx.Task.Variables
		reqMessage.MessageName = "RunNext"
		err := client.Message.SendMessage(&reqMessage)
		if err != nil {
			log.Printf("Error starting process: %s\n", err)
			return err
		}

		return err
	}
	return nil
}

// check if there's a 'next' process.
func getNext(newVars map[string]camundaclientgo.Variable, contx *processor.Context) error {
	fmt.Println("Getting next process ...")
	garbage := getUrl(camundaServer + "/process-instance?businessKeyLike=SentimentAnalysis&processDefinitionKey=SentimentHandler&variable=queue_eg_true")
	if garbage == nil {
		log.Fatal("GET URL Failed for unknown reasons")
	}
	if len(garbage) >= 1 {
		varb := contx.Task.Variables
		varb["foundNext"] = camundaclientgo.Variable{Value: "true", Type: "boolean"}
		err := contx.Complete(processor.QueryComplete{Variables: &varb})
		return err
	} else {
		varb := contx.Task.Variables
		varb["foundNext"] = camundaclientgo.Variable{Value: "false", Type: "boolean"}
		err := contx.Complete(processor.QueryComplete{Variables: &varb})
		return err
	}
}

func runAnalysis(newVars map[string]camundaclientgo.Variable, contx *processor.Context) error {

	var avg_sent float32 = 0.00
	var high_sent float32 = -100
	var low_sent float32 = 100
	var high_post string = ""
	var low_post string = ""
	var highURL string = ""
	var lowURL string = ""
	sent_num := 0
	thisQ, err := json.Marshal(contx.Task.Variables["searchterm"].Value)
	if err != nil {
		log.Fatal(err)
	}
	formValues := url.Values{}
	formValues.Set("query", string(thisQ))
	var DefaultClient = &http.Client{}

	urlPlace := ""
	myKey := ""
	if submitter.Forum == "platform" {
		urlPlace = PlatformURL
		myKey = PlatformAPIKey
	} else if submitter.Forum == "Cloud" {
		urlPlace = CloudURL
		myKey = CloudAPIKey
	} else if submitter.Forum == "BPMN.io" {
		urlPlace = BPMNUrl
		myKey = BPMNAPIKey
	}
	err = extendLock(contx, int(5 * time.Second))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", urlPlace, strings.NewReader(formValues.Encode()))
	if err != nil {
		fmt.Println("Request Object Failure")
		return err
	}
	request.Header.Set("Content-Type", "multipart/form-data")
	request.Header.Set("Api-Key", myKey)
	request.Header.Set("Api-Username", APIUser)
	request.Header.Set("Accept", "application/json")

	res, err := DefaultClient.Do(request)
	if err != nil {
		fmt.Println("HTTP GET Failed!", urlPlace)
		return err
	}
	if res.StatusCode != 200 {
		fmt.Println("Got other than Code 200")
		fmt.Println("Code: ", res.StatusCode)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	res.Body.Close()
	garbage := queryResponse{}
	// var prettyJSON bytes.Buffer
	// _ = json.Indent(&prettyJSON, data, "", "\t")
	// fmt.Println(string(prettyJSON.Bytes()))
	_ = json.Unmarshal(data, &garbage)

	ctx := context.Background()
	client, err := language.NewClient(ctx, option.WithCredentialsFile("credentials.json"))
	if err != nil {
		log.Fatal(err)
	}
	for x := 0; x < 10; x++ { // go the length of garbage
		err = extendLock(contx, int(7 * time.Second))
		if err != nil {
			return err
		}
		sentiment, err := analyzeSentiment(ctx, client, garbage.Rows[x][1])
		if err != nil {
			log.Fatal(err)
		}
		pId := garbage.Relations.Post[x].ID
		tId := garbage.Relations.Post[x].TopicID

		fmt.Printf("ID: %d TopicID: %d\n", pId, tId)

		reg := regexp.MustCompile(`[\n]`) // fix bolds (**foo**)
		translated := string(reg.ReplaceAll([]byte(garbage.Rows[x][1]), []byte("")))
		reg = regexp.MustCompile(`[']`)
		translated = string(reg.ReplaceAll([]byte(translated), []byte("\\'")))
		reg = regexp.MustCompile(`[\t]`)
		translated = string(reg.ReplaceAll([]byte(translated), []byte(" ")))
		// reg = regexp.MustCompile(`[,]`)
		// translated = string(reg.ReplaceAll([]byte(translated), []byte("\\,")))
		avg_sent += sentiment.DocumentSentiment.Score
		sent_num++
		if sentiment.DocumentSentiment.Score >= 0 {
			if sentiment.DocumentSentiment.Score > high_sent {
				high_sent = sentiment.DocumentSentiment.Score
				high_post = garbage.Relations.Post[x].Excerpt
				highURL = fmt.Sprintf("%s/t/%d", urlPlace, garbage.Relations.Post[x].TopicID)
			}
		} else {
			if sentiment.DocumentSentiment.Score < low_sent {
				low_sent = sentiment.DocumentSentiment.Score
				low_post = translated
				lowURL = fmt.Sprintf("%s/t/%d", urlPlace, garbage.Relations.Post[x].TopicID)
			}
		}
		time.Sleep(1 * time.Second)
	}
	avg_sent = avg_sent / float32(sent_num)
	fmt.Printf("Average: %.2f\nLow score: %.2f\nLow Post: %s\nHigh score: %.2f\nHigh post: %s\n", avg_sent, low_sent, low_post, high_sent, high_post)
	err = sendEmail(avg_sent, low_sent, low_post, lowURL, high_sent, high_post, highURL)
	if err != nil {
		return err
	}
	err = contx.Complete(processor.QueryComplete{Variables: &contx.Task.Variables})
	return err
}

func extendLock(contx *processor.Context, extTime int) error {
	var DefaultClient = &http.Client{}
	extender := make(map[string]interface{})
	extender["newDuration"] = extTime
	extender["workerId"] = "SentimentAnalyzer"
	data, err := json.Marshal(extender)
	exUrl := fmt.Sprintf("%s/external-task/%s/extendLock", camundaServer, contx.Task.Id)
	request, err := http.NewRequest("POST", exUrl, bytes.NewReader(data))
	if err != nil {
		fmt.Println("Request Object Failure")
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	res, err := DefaultClient.Do(request)
	if err != nil {
		fmt.Println("HTTP GET Failed!", exUrl)
		return err
	}
	if res.StatusCode != 204 {
		fmt.Println("Runner Got other than Code 204")
		fmt.Println("Code: ", res.StatusCode)
		return nil
		// log. Fatal(res.StatusCode)
	}
	return nil
}

// Check to see if any sentiment analysis is going on. If it is, park this process until it's done.
func queueStatus(newVars map[string]camundaclientgo.Variable, contx *processor.Context) error {
	fmt.Println("Checking the queue ... ")

	garbage := getUrl(camundaServer + "/process-instance?businessKeyLike=SentimentAnalysis&processDefinitionKey=runAnalysis")
	if garbage == nil {
		log.Fatal("Getting data failed for unknown reasons")
	}
	if len(garbage) == 0 { // no running instances
		varb := contx.Task.Variables
		varb["queue"] = camundaclientgo.Variable{Value: "false", Type: "boolean"}
		err := contx.Complete(processor.QueryComplete{Variables: &varb})
		return err
	} else {
		varb := contx.Task.Variables
		varb["queue"] = camundaclientgo.Variable{Value: "true", Type: "boolean"}
		err := contx.Complete(processor.QueryComplete{Variables: &varb})
		return err
	}
}

type smtpAuthentication struct {
	Server         string
	Port           int
	SenderEmail    string
	SenderIdentity string
	SMTPUser       string
	SMTPPassword   string
}

// sendOptions are options for sending an email
type sendOptions struct {
	To      string
	Subject string
}

// send sends the email
func send(smtpConfig smtpAuthentication, options sendOptions, htmlBody string, txtBody string) error {

	if smtpConfig.Server == "" {
		return errors.New("SMTP server config is empty")
	}
	if smtpConfig.Port == 0 {
		return errors.New("SMTP port config is empty")
	}

	if smtpConfig.SMTPUser == "" {
		return errors.New("SMTP user is empty")
	}

	if smtpConfig.SenderIdentity == "" {
		return errors.New("SMTP sender identity is empty")
	}

	if smtpConfig.SenderEmail == "" {
		return errors.New("SMTP sender email is empty")
	}

	if options.To == "" {
		return errors.New("no receiver emails configured")
	}

	from := mail.Address{
		Name:    smtpConfig.SenderIdentity,
		Address: smtpConfig.SenderEmail,
	}

	m := gomail.NewMessage()
	m.SetHeader("From", from.String())
	m.SetHeader("To", options.To)
	m.SetHeader("Subject", options.Subject)

	m.SetBody("text/plain", txtBody)
	m.AddAlternative("text/html", htmlBody)

	d := gomail.NewDialer(smtpConfig.Server, smtpConfig.Port, smtpConfig.SMTPUser, smtpConfig.SMTPPassword)

	return d.DialAndSend(m)
}

const letterBytes = "+-=abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytesRmndr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}


type SentimentTheme struct{
	Term string
}

func (dt *SentimentTheme) Name() string {
	return "Discourse Sentiment Theme"
}

func (dt *SentimentTheme) HTMLTemplate() string {
    // Get the template from a file (if you want to be able to change the template live without retstarting your application)
    // Or write the template by returning pure string here (if you want embbeded template and do not bother with external dependencies)
		contents, err := ioutil.ReadFile("./static/email.html")
		if err != nil {
			log.Fatal(err)
		}
		return string(contents)

}

func (dt *SentimentTheme) PlainTextTemplate() string {
    // Get the template from a file (if you want to be able to change the template live without retstarting your application)
    // Or write the template by returning pure string here (if you want embbeded template and do not bother with external dependencies)
		contents, err := ioutil.ReadFile("./static/email.txt")
		if err != nil {
			log.Fatal(err)
		}
		return string(contents)
}

// h := hermes.Hermes{
//     Theme: new(SentimentTheme) // Set your fresh new theme here
//     Product: hermes.Product{
//         Name: "Hermes",
//         Link: "https://example-hermes.com/",
//     },
// }
