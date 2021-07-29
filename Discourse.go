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
	"github.com/go-gomail/gomail"
	"github.com/matcornic/hermes/v2"
	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/api/option"
	languagepb "google.golang.org/genproto/googleapis/cloud/language/v1"
)

const secret string = "camundaBPMDiscourseTester"

// APIKey is your Discourse API_KEY here
const PlatformAPIKey = "0476e2a15a1152d8c00c1eeec60bbcec4977e54b87f2d7556a857e814c5a4fb0"
const PlatformURL = "https://forum.camunda.org/admin/plugins/explorer/queries/9/run"
const BPMNAPIKey = "d909293c78ee5cc6fa055419da5f9682a595341a2eb1acb02ce17d7bcb7c060c"
const BPMNUrl = "https://forum.bpmn.io/admin/plugins/explorer/queries/2/run"

const CloudAPIKey = ""
const CloudURL = "https://forum.camunda.io"

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
		http.Redirect(w, r, "https://davidgs.com:9999/thanks.html", http.StatusSeeOther)
		w.WriteHeader(200)
		qString := html.UnescapeString(string(data))
		fmt.Println(qString)
		foo := strings.Split(qString, "&")
		for x := 0; x < len(foo); x++ {
			fmt.Println(foo[x])
			if strings.HasPrefix(foo[x], "username") {
				submitter.Username = strings.ReplaceAll(strings.Split(foo[x], "=")[1], "+", " ")
			} else if strings.HasPrefix(foo[x], "emailAddress") {
				submitter.Email = strings.ReplaceAll(strings.Split(foo[x], "=")[1], "%40", "@")
			} else if strings.HasPrefix(foo[x], "community") {
				submitter.Forum = strings.Split(foo[x], "=")[1]
			} else if strings.HasPrefix(foo[x], "searchterm") {
				submitter.SearchTerm = strings.ReplaceAll(strings.Split(foo[x], "=")[1], "+", " ")
			} else {

			}
		}
		runQuery(submitter.SearchTerm)
	}
}

func runQuery(query string) {
	var avg_sent float32 = 0.00
	var high_sent float32 = -100
	var low_sent float32 = 100
	var high_post string = ""
	var low_post string = ""
	var highURL string = ""
	var lowURL string = ""
	sent_num := 0
	fmt.Printf("Running Query: 9\n")
	formValues := url.Values{}
	if len(query) <= 0 {
		query = "connectors"
	}
	formValues.Set("query", query)
	var DefaultClient = &http.Client{}

	urlPlace := ""
	myKey := ""
	if submitter.Forum == "platform" {
		urlPlace = PlatformURL
		myKey = PlatformAPIKey
	} else if submitter.Forum == "Cloud" {

	} else if submitter.Forum == "BPMN.io" {
		urlPlace = BPMNUrl
		myKey = BPMNAPIKey
	}
	request, err := http.NewRequest("POST", urlPlace, strings.NewReader(formValues.Encode()))
	if err != nil {
		fmt.Println("Request Object Failure")
		log.Fatal(err)
	}
	request.Header.Set("Content-Type", "multipart/form-data")
	request.Header.Set("Api-Key", myKey)
	request.Header.Set("Api-Username", APIUser)
	request.Header.Set("Accept", "application/json")

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
	var prettyJSON bytes.Buffer
	_ = json.Indent(&prettyJSON, data, "", "\t")
	fmt.Println(string(prettyJSON.Bytes()))
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

	for x := 0; x < 10; x++ { //len(garbage.Rows); x++ {
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
	h := hermes.Hermes{
		// Optional Theme
		// Theme: new(Default),
		Product: hermes.Product{
			// Appears in header & footer of e-mails
			Name: "Camunda, Inc.",
			Link: "https://camunda.com/",
			// Optional product logo
			Logo: "https://camunda.com/&psig=AOvVaw1XnYL6hcEfkaCqzwUue-BT&ust=1625951056898000&source=images&cd=vfe&ved=0CAoQjRxqFwoTCNDJuIfy1vECFQAAAAAdAAAAABAD",
		},
	}
	email := hermes.Email{
		Body: hermes.Body{
			Name: submitter.Username,
			Intros: []string{
				"Your Sentiment Analysis is complete!",
			},
			// Actions: []hermes.Action{
			//     {
			//         Instructions: "To get started with Hermes, please click here:",
			//         Button: hermes.Button{
			//             Color: "#22BC66", // Optional action button color
			//             Text:  "Confirm your account",
			//             Link:  "https://hermes-example.com/confirm?token=d9729feb74992cc3482b350163a1a010",
			//         },
			//     },
			// },
			Outros: []string{
				"Here's what we found in analysing the term 'connectors'",
				"The average sentiment was: ",
				fmt.Sprintf("%f", avg_sent),
				"The high sentiment was: ",
				fmt.Sprintf("%f", high_sent),
				"Which was ",
				high_post,
				"You can read the whole post here: ",
				highURL,
				"The low sentiment was: ",
				fmt.Sprintf("%f", low_sent),
				"which was ",
				low_post,
				"And you can read that post here: ",
				lowURL,
			},
		},
	}

	// Generate an HTML email with the provided contents (for modern clients)
	emailBody, err := h.GenerateHTML(email)
	if err != nil {
		panic(err) // Tip: Handle error with something else than a panic ;)
	}

	// Generate the plaintext version of the e-mail (for clients that do not support xHTML)
	emailText, err := h.GeneratePlainText(email)
	if err != nil {
		panic(err) // Tip: Handle error with something else than a panic ;)
	}

	// Optionally, preview the generated HTML e-mail by writing it to a local file
	err = ioutil.WriteFile("preview.html", []byte(emailBody), 0644)
	if err != nil {
		panic(err) // Tip: Handle error with something else than a panic ;)
	}
	sendEmails := os.Getenv("HERMES_SEND_EMAILS") == "true"
	if sendEmails {
		port, _ := strconv.Atoi(os.Getenv("HERMES_SMTP_PORT"))
		password := os.Getenv("HERMES_SMTP_PASSWORD")
		SMTPUser := os.Getenv("HERMES_SMTP_USER")
		if password == "" {
			fmt.Printf("Enter SMTP password of '%s' account: ", SMTPUser)
			bytePassword, _ := terminal.ReadPassword(0)
			password = string(bytePassword)
		}
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
		options.Subject = "Camunda Sentiment"
		fmt.Printf("Sending email '%s'...\n", options.Subject)
		err = send(smtpConfig, options, string(emailBody), string(emailText))
		if err != nil {
			panic(err)
		}
	}
	f.Close()
	avg_sent = avg_sent / float32(sent_num)
	fmt.Printf("Average: %f2\nLow score: %f2\nLow Post: %s\nHigh score: %f2\nHigh post: %s\n", avg_sent, low_sent, low_post, high_sent, high_post)
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
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/sentiment", topicEvent)
	// // http.Handle("/test/", http.StripPrefix("/test", fs)) // set router
	err := http.ListenAndServeTLS(":9999", "/home/davidgs/.node-red/combined", "/home/davidgs/.node-red/combined", nil)
	if err != nil {
		log.Fatal(err)
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
