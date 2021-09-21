package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/mail"
	"os"
	"regexp"

	// "regexp"
	"strings"
	"time"

	// [START external imports]
	language "cloud.google.com/go/language/apiv1"
	camundaclientgo "github.com/citilinkru/camunda-client-go/v2"
	"github.com/citilinkru/camunda-client-go/v2/processor"
	"github.com/go-gomail/gomail"
	"github.com/matcornic/hermes/v2"

	// "github.com/opentracing/opentracing-go/log"
	"github.com/russross/blackfriday/v2"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	languagepb "google.golang.org/genproto/googleapis/cloud/language/v1"
	"gopkg.in/yaml.v2"
)

// Submitter is where we keep track of all the info about who submitted what.
type Submitter struct {
	Username      string `json:"username"`
	Email         string `json:"emailAddress"`
	Forum         string `json:"community"`
	SearchTerm    string `json:"searchterm"`
	ThemeTemplate string `json:"themeTemplate,omitempty"`
	ThemeTemp     string `json:"themeTemp,omitempty"`
	Exact         string `json:"exact,omitempty"`
	Total				 int    `json:"total,omitempty"`
}

// ServerSettings is all the stuff we need for the server to run, but don't
// want to hard-code. Because hard-coding is bad.
type ServerSettings struct {
	EmailSettings struct {
		SEND_EMAILS     bool   `yaml:"sendmail"`
		SMTP_PORT       int    `yaml:"smtp_port"`
		SMTP_PASSWORD   string `yaml:"passwd"`
		SMTP_USER       string `yaml:"username"`
		SMTP_SERVER     string `yaml:"server"`
		SENDER_EMAIL    string `yaml:"email"`
		SENDER_IDENTITY string `yaml:"identity"`
	} `yaml:"Email Settings"`
	CamundaHost struct {
		Name     string `yaml:"name"`
		Port     int    `yaml:"port"`
		Protocol string `yaml:"protocol"`
		Host     string `yaml:"host"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"Camunda Host"`
	ForumSettings []struct {
		Name    string `yaml:"name"`
		URL     string `yaml:"serverUrl"`
		APIKey  string `yaml:"apikey"`
		APIUser string `yaml:"apiuser"`
		Exact   int    `yaml:"exact"`
	} `yaml:"Forum Settings"`
}

var serverSettings = ServerSettings{}



// ProcessQueryResult is what we get from a process query to Camunda Engine
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

// CamundaRestError is an error implementation that includes a time and message.
type CamundaRestError struct {
	When time.Time
	What string
}

func (e CamundaRestError) Error() string {
	return fmt.Sprintf("%v: %v", e.When, e.What)
}

// apologies. I _hate_ global vars, but I didn't have time to refactor everything
// to make this work any other way.
var ThemeTemp = ""

// Run google Sentiment Analysis on the given string
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

// recieve incoming form requests.
// We don't allow GETs
func topicEvent(w http.ResponseWriter, r *http.Request) {
	log.Debug("Incoming message ... ")
	if r.Method == "GET" {
		// you get nothing.
		http.Error(w, "GET Method not supported", 400)
		log.Warn("topicEvent: Rejected HTTP GET from ", r.Header.Get("X-Forwarded-For"))
		return
	} else {
		var submitter = Submitter{}
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error("topicEvent: Got no Data for form responses", err)
			return
		}
		r.Body.Close()
		qString := html.UnescapeString(string(data))
		formFields := strings.Split(qString, "&")
		for _, v := range formFields {
			if strings.Contains(v, "=") {
				split := strings.Split(v, "=")
				switch split[0] {
				case "username":
					submitter.Username = strings.ReplaceAll(split[1], "+", " ")
				case "emailAddress":
					submitter.Email = strings.ReplaceAll(split[1], "%40", "@")
				case "searchterm":
					submitter.SearchTerm = strings.ReplaceAll(split[1], "+", " ")
				case "community":
					submitter.Forum = split[1]
				case "exact":
					submitter.Exact = split[1]
					if submitter.Exact == "off" {
						submitter.SearchTerm = strings.ReplaceAll(submitter.SearchTerm, " ", " AND ")
					} else {
						submitter.SearchTerm = fmt.Sprintf("\"%s\"", submitter.SearchTerm)
					}
				default:
				}
			}
		}
		submitter.ThemeTemplate = "./static/email"
		// We only allow camunda employees, with their Camunda email address.
		if !strings.HasSuffix(submitter.Email, "camunda.com") {
			http.Redirect(w, r, "https://sentiment.camunda.com/sorry.html", http.StatusSeeOther)
			log.Errorf("Outside email address: %s, IP: ", submitter.Email, r.Header.Get("X-Forwarded-For"))
			return
		}
		// be polite.
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
		variables["exact"] = camundaclientgo.Variable{
			Value: submitter.Exact,
			Type:  "string",
		}
		// businessKey has to be unique, so me make it (mostly) unique
		businessKey := fmt.Sprintf("SentimentAnalysis-%s", RandString(8))
		log.Debugf("Using BusinessKey: %s\n", businessKey)
		opts := camundaclientgo.ClientOptions{}
		opts.EndpointUrl = serverSettings.CamundaHost.Protocol + "://" + serverSettings.CamundaHost.Host + ":" + fmt.Sprintf("%d", serverSettings.CamundaHost.Port) + "/engine-rest"
		opts.Timeout = time.Second * 20
		opts.ApiUser = serverSettings.CamundaHost.User
		opts.ApiPassword = serverSettings.CamundaHost.Password
		client := camundaclientgo.NewClient(opts)
		reqMessage := camundaclientgo.ReqMessage{}
		reqMessage.BusinessKey = businessKey
		reqMessage.ProcessVariables = &variables
		reqMessage.MessageName = "submit_form"
		err = client.Message.SendMessage(&reqMessage)
		if err != nil {
			log.Errorf("topicEvent: Error starting process: %s\n", err)
			return
		}
	}
	log.Debug("Form Handling complete.")
}

// Actually send the email ...
func sendEmail(avg_sent float32, low_sent float32, low_post string, lowURL string, high_sent float32, high_post string, highURL string, submitter Submitter) error {
	log.Debugf("Sending email to: %s\n", submitter.Email)
	h := hermes.Hermes{
		Theme: new(SentimentTheme),
		Product: hermes.Product{
			// Appears in header & footer of e-mails
			Name: "Camunda, Inc.",
			Link: "https://sentiment.camunda.com/",
			// Optional product logo
			Logo:      "https://sentiment.camunda.com/images/Logo_White.svg",
			Copyright: "© 2021 Camunda, Inc.",
		},
	}
	forum := ""
	if submitter.Forum == "platform" {
		forum = "<a href=\"https://forum.camunda.org/\">Platform</a>"
	}
	if submitter.Forum == "cloud" {
		forum = "<a href=\"https://forum.camunda.io\">Cloud</a>"
	}
	if submitter.Forum == "bpmn" {
		forum = "<a href=\"https://forum.bpmn.io/\">BPMN</a>"
	}
	// This is the body of our email ...
	ft := ""
	if submitter.Exact == "on" {
		ft = fmt.Sprintf("Here is what we found in analyzing the exact phrase `%s` in the %s forum", submitter.SearchTerm, forum)
	} else {
		ft = fmt.Sprintf("Here is what we found in analyzing the phrase `%s` in the %s forum.", submitter.SearchTerm, forum)
	}
	fm2 := ft + `

The search returned ` + fmt.Sprintf("%d", submitter.Total) + ` results.`
	if submitter.Total > 0 {
		fm2 += `
The overall average sentiment was: 	**` + fmt.Sprintf("%.2f", avg_sent) + `**

The high sentiment was: 		**` + fmt.Sprintf("%.2f", high_sent) + `**

Here's an excerpt from that post:

> ` + high_post + `

You can read the whole post [here](` + highURL + `)

The low sentiment was: 		**` + fmt.Sprintf("%.2f", low_sent) + `**

Here's an excerpt from that post:

> ` + low_post + `

You can read the whole post [here](` + lowURL + `).
`
	} else {
		fm2 += `
You might try broadening your search terms in order to get more results
`
	}
	fm2 += `

## Data Submitted:
- Submitter Name:	` + submitter.Username + `
- Submitter Email:	` + submitter.Email + `
- Forum Searched:	` + submitter.Forum + `
- Search Term:	` + submitter.SearchTerm + `
`

	// This whole thing is a hack because Hermes templates are so unbelievably limited
	// in usefulness. So we basically are only using Hermes for parts of the email process.
	// Now we use BlackFriday to turn this Markdown string into an html string
	output := blackfriday.Run([]byte(fm2), blackfriday.WithExtensions(blackfriday.LaxHTMLBlocks))
	log.Debug("Message: ", string(output))
	contents, err := ioutil.ReadFile(submitter.ThemeTemplate + ".html")
	if err != nil {
		log.Error("sendEmail: ", err)
		return err
	}
	// hacky kludge has entered the chat
	newFile := strings.Replace(string(contents), "{{FILL_IN_THIS}}", string(output), -1)
	fl := "./temp-email/" + submitter.Email
	err = ioutil.WriteFile(fl+".html", []byte(newFile), 0644)
	if err != nil {
		log.Error("sendEmail: ", err)
		return err
	}
	contents, err = ioutil.ReadFile(submitter.ThemeTemplate + ".txt")
	if err != nil {
		log.Error("sendEmail: ", err)
		return err
	}
	newFile = strings.Replace(string(contents), "{{FILL_IN_THIS}}", string(output), -1)
	fl = "./temp-email/" + submitter.Email
	err = ioutil.WriteFile(fl+".txt", []byte(newFile), 0644)
	if err != nil {
		log.Error("sendEmail: ", err)
		return err
	}
	ThemeTemp = fl
	// end of awful hack.
	// we now continue with our regularly scheduled Hermes email
	email := hermes.Email{
		Body: hermes.Body{
			Name:   submitter.Username,
			Intros: []string{string(output)},
		},
	}

	// Generate an HTML email with the provided contents (for modern clients)
	emailBody, err := h.GenerateHTML(email)
	if err != nil {
		log.Error("sendEmail: ", err)
		ThemeTemp = ""
		return err
	}

	// Generate the plaintext version of the e-mail (for clients that do not support xHTML)
	emailText, err := h.GeneratePlainText(email)
	if err != nil {
		log.Error("sendEmail: ", err)
		ThemeTemp = ""
		return err
	}

	if serverSettings.EmailSettings.SEND_EMAILS {
		smtpConfig := smtpAuthentication{
			Server:         serverSettings.EmailSettings.SMTP_SERVER,
			Port:           serverSettings.EmailSettings.SMTP_PORT,
			SenderEmail:    serverSettings.EmailSettings.SENDER_EMAIL,
			SenderIdentity: serverSettings.EmailSettings.SENDER_IDENTITY,
			SMTPPassword:   serverSettings.EmailSettings.SMTP_PASSWORD,
			SMTPUser:       serverSettings.EmailSettings.SMTP_USER,
		}
		options := sendOptions{
			To: submitter.Email,
		}
		options.Subject = "Camunda Sentiment Analysis Results"
		err = send(smtpConfig, options, string(emailBody), string(emailText))
		if err != nil {
			log.Error("sendEmail: ", err)
			return err
		}
	}
	err = os.Remove(fl + ".html")
	if err != nil {
		log.Warn("sendEmail: ", err)
		return err
	}
	err = os.Remove(fl + ".txt")
	if err != nil {
		log.Warn("sendEmail: ", err)
		return err
	}
	log.Debug("Email sent!")
	return nil
}

func main() {
	logPtr := flag.String("log", "Debug", "LogLevel, accepts Debug, Info, Warn, or Error")
	fPtr := flag.String("file", "stdout", "Log File Location, or stdout for console logging")
	flag.Parse()
	if *logPtr == "Debug" {
		log.SetLevel(log.DebugLevel)
	} else if *logPtr == "Info" {
		log.SetLevel(log.InfoLevel)
	} else if *logPtr == "Warn" {
		log.SetLevel(log.WarnLevel)
	} else if *logPtr == "Error" {
		log.SetLevel(log.ErrorLevel)
	} else if *logPtr == "Trace" {
		log.SetLevel(log.TraceLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}
	if *fPtr != "stdout" {
		f, err := os.OpenFile(*fPtr, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		log.SetOutput(f)
	} else {
		log.SetOutput(os.Stdout)
	}
	dat, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		log.Fatal("No startup file: ", err)
	}
	err = yaml.Unmarshal(dat, &serverSettings)
	if err != nil {
		log.Fatal("Config file not a yaml file ", err)
	}
	log.Debug("Starting up!")
	client := camundaclientgo.NewClient(camundaclientgo.ClientOptions{EndpointUrl: serverSettings.CamundaHost.Protocol + "://" + serverSettings.CamundaHost.Host + ":" + fmt.Sprintf("%d", serverSettings.CamundaHost.Port) + "/engine-rest",
		Timeout:     time.Second * 20,
		ApiUser:     serverSettings.CamundaHost.User,
		ApiPassword: serverSettings.CamundaHost.Password,
	})
	logger := func(err error) {
		log.Error(err)
	}
	asyncResponseTimeout := 5000
	// get a process instance to work with

	proc := processor.NewProcessor(client, &processor.ProcessorOptions{
		WorkerId:                  "SentimentAnalyzer",
		LockDuration:              time.Second * 20,
		MaxTasks:                  10,
		MaxParallelTaskPerHandler: 100,
		LongPollingTimeout:        20 * time.Second,
		AsyncResponseTimeout:      &asyncResponseTimeout,
	}, logger)
	log.Debug("Processor started ... ")
	// add a handler for checking the existing Queue
	proc.AddHandler(
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "checkQueue"},
		},
		func(ctx *processor.Context) error {
			return queueStatus(ctx.Task.Variables, ctx)
		},
	)
	log.Debug("checkQueue Handler started ... ")
	// Handler for running a full sentiment analysis
	proc.AddHandler(
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "runThisAnalysis"},
		},
		func(ctx *processor.Context) error {
			return runAnalysis(ctx.Task.Variables, ctx)
		},
	)
	log.Debug("runAnalysis Handler started ... ")
	// Handler to check for waiting processes
	proc.AddHandler(
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "checkForNext"},
		},
		func(ctx *processor.Context) error {
			return getNext(ctx.Task.Variables, ctx)
		},
	)
	log.Debug("checkForNext Handler started ... ")
	// process to fire off the next sentiment analysis
	proc.AddHandler(
		&[]camundaclientgo.QueryFetchAndLockTopic{
			{TopicName: "fireNext"},
		},
		func(ctx *processor.Context) error {
			return fireNext(ctx.Task.Variables, ctx)
		},
	)
	log.Debug("fireNext Handler started ... ")
	http.HandleFunc("/sentiment", topicEvent)
	// http.Handle("/", http.FileServer(http.Dir("./static")))
	err = http.ListenAndServeTLS(":9999", "./cert1.pem", "./privkey1.pem", nil)
	if err != nil {
		log.Fatal(err)
	}
}

// run a `GET` request against the specified Camunda Engine URL and return the results
func getUrl(url string) ProcessQueryResult {
	log.Debugf("getting URL %s", url)
	var DefaultClient = &http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	request.SetBasicAuth(serverSettings.CamundaHost.User, serverSettings.CamundaHost.Password)
	if err != nil {
		log.Error("getUrl: ", err)
	}
	res, err := DefaultClient.Do(request)
	if err != nil {
		log.Error("getUrl: ", err)
	}
	if res.StatusCode >= 220 {
		log.Warnf("getUrl: URL: %s : Status: %d", url, res.StatusCode)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("getUrl: Got no Data from %s", url)
	}
	log.Debugf("Got data: %s", string(data))
	res.Body.Close()
	garbage := ProcessQueryResult{}
	err = json.Unmarshal(data, &garbage)
	if err != nil {
		log.Error("getUrl: ", err)
	}
	return garbage
}

// fire of fthe next waiting process
func fireNext(newVars map[string]camundaclientgo.Variable, contx *processor.Context) error {
	log.Debug("Firing Next Process")
	// complete the process first, else it shows up as still active and all hell breaks loose
	err := contx.Complete(processor.QueryComplete{Variables: &contx.Task.Variables})
	if err != nil {
		log.Error("Fire next: ", err)
		return err
	}
	// get a list of waiting processes
	cURL := serverSettings.CamundaHost.Protocol + "://" + serverSettings.CamundaHost.Host + ":" + fmt.Sprintf("%d", serverSettings.CamundaHost.Port) + "/engine-rest" + "/process-instance?businessKeyLike=SentimentAnalysis&processDefinitionKey=SentimentHandler&variable=queue_eg_true"
	garbage := getUrl(cURL)
	if garbage == nil {
		log.Error("fireNext: Failed to get next for unknown reasons")
	}
	if len(garbage) >= 1 { // there's at least one process waiting
		log.Debug("Found a waiting process ...")
		opts := camundaclientgo.ClientOptions{}
		opts.EndpointUrl = serverSettings.CamundaHost.Protocol + "://" + serverSettings.CamundaHost.Host + ":" + fmt.Sprintf("%d", serverSettings.CamundaHost.Port) + "/engine-rest"
		opts.Timeout = time.Second * 20
		opts.ApiUser = serverSettings.CamundaHost.User
		opts.ApiPassword = serverSettings.CamundaHost.Password
		client := camundaclientgo.NewClient(opts)
		reqMessage := camundaclientgo.ReqMessage{}
		reqMessage.BusinessKey = garbage[0].BusinessKey
		reqMessage.ProcessVariables = &contx.Task.Variables
		reqMessage.MessageName = "RunNext"
		err := client.Message.SendMessage(&reqMessage)
		if err != nil {
			log.Warnf("fireNext: Error starting process: %s\n", err)
			return err
		}
	}
	log.Debug("Fire Next Complete")
	return nil
}

// check if there's a 'next' process.
func getNext(newVars map[string]camundaclientgo.Variable, contx *processor.Context) error {
	log.Debug("Getting next ...")
	// cClient := camundaclientgo.NewClient(camundaclientgo.ClientOptions{EndpointUrl: serverSettings.CamundaHost.Protocol + "://" + serverSettings.CamundaHost.Host + ":" + fmt.Sprintf("%d", serverSettings.CamundaHost.Port) + "/engine-rest",
	// 	Timeout: time.Second * 20,
	// })
	// qParams := make(map[string]string)
	// qParams["businessKeyLike"]="SentimentAnalysis"
	// qParams["processDefinitionKey"]="SentimentHandler"
	// qParams["variable"]="queue_eg_true"
	// pParams := make(map[string]string)
	// pParams["businessKeyLike"]="SentimentAnalysis"
	// pParams["processDefinitionKey"]="SentimentHandler"
	// pParams["latestVersion"]="true"
	// pDefs, err := cClient.ProcessDefinition.GetList(pParams)
	// if err != nil {
	// 	log.Error(err)
	// }
	// garbage := pDefs
	// exProc := ""
	// for _, t := range pDefs {
	// 	if t.Name == "SentimentHandler" {
	// 		exProc = t.Id
	// 		break
	// 	}
	// }
	// if exProc != "" {
	// cClient.ProcessInstance.GetList(qParams)
	// }

	// if len(garbage) >= 1 {
	// 	varb := contx.Task.Variables
	// 	varb["foundNext"] = camundaclientgo.Variable{Value: "true", Type: "boolean"}
	// 	err := contx.Complete(processor.QueryComplete{Variables: &varb})
	// 	if err != nil {
	// 		log.Error("getNext: ", err)
	// 		return err
	// 	}
	// } else {
	// 	varb := contx.Task.Variables
	// 	varb["foundNext"] = camundaclientgo.Variable{Value: "false", Type: "boolean"}
	// 	err := contx.Complete(processor.QueryComplete{Variables: &varb})
	// 	if err != nil {
	// 		log.Error("getNext: ", err)
	// 		return err
	// 	}
	// }
	surl := serverSettings.CamundaHost.Protocol + "://" + serverSettings.CamundaHost.Host + ":" + fmt.Sprintf("%d", serverSettings.CamundaHost.Port) + "/engine-rest" + "/process-instance?businessKeyLike=SentimentAnalysis&processDefinitionKey=SentimentHandler&variable=queue_eg_true"
	garbage := getUrl(surl)
	if garbage == nil {
		e := CamundaRestError{When: time.Now().UTC(), What: "Got no content from Camunda Engine"}
		log.Errorf("getNext: GET URL %s Failed for unknown reasons", surl)
		return e
	}
	if len(garbage) >= 1 {
		log.Debug("Found a process in the queue ...")
		varb := contx.Task.Variables
		varb["foundNext"] = camundaclientgo.Variable{Value: "true", Type: "boolean"}
		err := contx.Complete(processor.QueryComplete{Variables: &varb})
		if err != nil {
			log.Error("getNext: ", err)
			return err
		}
	} else {
		varb := contx.Task.Variables
		varb["foundNext"] = camundaclientgo.Variable{Value: "false", Type: "boolean"}
		err := contx.Complete(processor.QueryComplete{Variables: &varb})
		if err != nil {
			log.Error("getNext: ", err)
			return err
		}
	}
	log.Debug("Get next complete")
	return nil
}

func runAnalysis(newVars map[string]camundaclientgo.Variable, contx *processor.Context) error {
	log.Debug("Running an analysis ... ")
	cClient := camundaclientgo.NewClient(camundaclientgo.ClientOptions{EndpointUrl: serverSettings.CamundaHost.Protocol + "://" + serverSettings.CamundaHost.Host + ":" + fmt.Sprintf("%d", serverSettings.CamundaHost.Port) + "/engine-rest",
		Timeout:     time.Second * 20,
		ApiUser:     serverSettings.CamundaHost.User,
		ApiPassword: serverSettings.CamundaHost.Password,
	})

	var avg_sent float32 = 0.00
	var high_sent float32 = -100
	var low_sent float32 = 100
	var high_post string = ""
	var low_post string = ""
	var highURL string = ""
	var lowURL string = ""
	sent_num := 0
	f, err := json.Marshal(newVars["searchterm"].Value)
	if err != nil {
		log.Error(err)
		return err
	}
	fmt.Println(string(f))
	thisQ := fmt.Sprintf("%s", newVars["searchterm"].Value)

	thisQ = fmt.Sprintf("?q=%s", thisQ)
	reg := regexp.MustCompile(`["]`)
	thisQ = string(reg.ReplaceAll([]byte(thisQ), []byte("%22")))
	reg = regexp.MustCompile(`[ ]`)
	thisQ = string(reg.ReplaceAll([]byte(thisQ), []byte("%20")))
	log.Debug("Search Term: ", thisQ)
	urlPlace := ""
	myKey := ""
	linker := ""
	if newVars["forum"].Value == "platform" {
		urlPlace = serverSettings.ForumSettings[0].URL
		myKey = serverSettings.ForumSettings[0].APIKey
	} else if newVars["forum"].Value == "cloud" {
		urlPlace = serverSettings.ForumSettings[1].URL
		myKey = serverSettings.ForumSettings[1].APIKey
	} else if newVars["forum"].Value == "bpmn" {
		urlPlace = serverSettings.ForumSettings[2].URL
		myKey = serverSettings.ForumSettings[2].APIKey
	}
	log.Debugf("Using Query URL: %s", urlPlace)
	linker = strings.Split(urlPlace, "search")[0]
	delay := int(2 * time.Second)
	ext := camundaclientgo.QueryExtendLock{
		NewDuration: &delay,
		WorkerId:    &contx.Task.WorkerId,
	}
	err = cClient.ExternalTask.ExtendLock(contx.Task.Id, ext)
	if err != nil {
		log.Error("runAnalysis::extendLock: ", err)
		return err
	}
	// log.Debug("Sending query ...")
	aq := NewAPIQuery(urlPlace, thisQ, myKey, "davidgs")
	err = aq.SendRequest()
	if err != nil {
		log.Error("runAnalysis::SendRequest: ", err)
		return err
	}
	fmt.Printf("Query returned %d Results\n", len(aq.APIResponse.Posts))
	ctx := context.Background()
	client, err := language.NewClient(ctx, option.WithCredentialsFile("credentials.json"))
	if err != nil {
		log.Error("runAnalysis::NewClient: ", err)
		return err
	}
	log.Debugf("Analyzing %d posts.\n", len(aq.APIResponse.Posts))
	for x := 0; x < len(aq.APIResponse.Posts); x++ {
		err = cClient.ExternalTask.ExtendLock(contx.Task.Id, ext)
		if err != nil {
			log.Error("runAnalysis: ", err)
			return err
		}
		sentiment, err := analyzeSentiment(ctx, client, fmt.Sprintf("%v", aq.APIResponse.Posts[x].Blurb))
		if err != nil {
			log.Error("runAnalysis: ", err)
			continue
		}
		log.Debugf("Post %d/%d ID: %d TopicID: %d\n", x, len(aq.APIResponse.Posts), aq.APIResponse.Posts[x].ID, aq.APIResponse.Posts[x].TopicID)
		log.Debugf("Post TopicId: %d Sentiment: %.2f Blurb: %s\n", aq.APIResponse.Posts[x].TopicID, sentiment.DocumentSentiment.Score, aq.APIResponse.Posts[x].Blurb)
		avg_sent += sentiment.DocumentSentiment.Score
		sent_num++
		if sentiment.DocumentSentiment.Score >= 0 {
			if sentiment.DocumentSentiment.Score > high_sent {
				high_sent = sentiment.DocumentSentiment.Score
				high_post = aq.APIResponse.Posts[x].Blurb
				highURL = fmt.Sprintf("%s/t/%d", linker, aq.APIResponse.Posts[x].TopicID)
			}
		} else {
			if sentiment.DocumentSentiment.Score < low_sent {
				low_sent = sentiment.DocumentSentiment.Score
				low_post = aq.APIResponse.Posts[x].Blurb
				lowURL = fmt.Sprintf("%s/t/%d", linker, aq.APIResponse.Posts[x].TopicID)
			}
		}
		time.Sleep(1 * time.Second)
	}
	err = contx.Complete(processor.QueryComplete{Variables: &contx.Task.Variables})
	if err != nil {
		log.Error("Cannot complete task: ", err)
		return err
	}
	log.Debugf("Link to article: %s", linker)
	var submitter = Submitter{}
	submitter.Email = fmt.Sprint(contx.Task.Variables["email"].Value)
	submitter.Username = fmt.Sprint(contx.Task.Variables["username"].Value)
	submitter.Forum = fmt.Sprint(contx.Task.Variables["forum"].Value)
	submitter.SearchTerm = fmt.Sprint(contx.Task.Variables["searchterm"].Value)
	submitter.Exact = fmt.Sprint(contx.Task.Variables["exact"].Value)
	submitter.ThemeTemplate = "./static/email"
	submitter.Total = len(aq.APIResponse.Posts)
	avg_sent = avg_sent / float32(sent_num)
	log.Debugf("Average: %.2f\nLow score: %.2f\nLow Post: %s\nHigh score: %.2f\nHigh post: %s\n", avg_sent, low_sent, low_post, high_sent, high_post)
	err = sendEmail(avg_sent, low_sent, low_post, lowURL, high_sent, high_post, highURL, submitter)
	if err != nil {
		log.Error("runAnalysis: ", err)
		return err
	}

	log.Debug("Analysis Complete.")
	return nil
}

func extendLock(contx *processor.Context, extTime int) error {
	var DefaultClient = &http.Client{}
	extender := make(map[string]interface{})
	extender["newDuration"] = extTime
	extender["workerId"] = "SentimentAnalyzer"
	data, _ := json.Marshal(extender)
	exUrl := fmt.Sprintf("%s/external-task/%s/extendLock", serverSettings.CamundaHost.Protocol+"://"+serverSettings.CamundaHost.Host+":"+fmt.Sprintf("%d", serverSettings.CamundaHost.Port)+"/engine-rest", contx.Task.Id)
	request, err := http.NewRequest("POST", exUrl, bytes.NewReader(data))
	if err != nil {
		log.Error("extendLock: ", err)
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	res, err := DefaultClient.Do(request)
	if err != nil {
		log.Error("extendLock: ", err)
		return err
	}
	if res.StatusCode != 204 {
		e := CamundaRestError{When: time.Now().UTC(), What: fmt.Sprintf("extendLock got status code: %d", res.StatusCode)}
		log.Error(e)
		return e
	}
	return nil
}

// Check to see if any sentiment analysis is going on. If it is, park this process until it's done.
func queueStatus(newVars map[string]camundaclientgo.Variable, contx *processor.Context) error {
	log.Debug("Checking the queue ... ")
	cURL := serverSettings.CamundaHost.Protocol + "://" + serverSettings.CamundaHost.Host + ":" + fmt.Sprintf("%d", serverSettings.CamundaHost.Port) + "/engine-rest" + "/process-instance?businessKeyLike=SentimentAnalysis&processDefinitionKey=runAnalysis"
	garbage := getUrl(cURL)
	if garbage == nil {
		e := CamundaRestError{When: time.Now().UTC(), What: fmt.Sprintf("queueStatus: %s returned empty data", cURL)}
		log.Error("queueStatus: ", e)
		return e
	}
	if len(garbage) == 0 { // no running instances
		varb := contx.Task.Variables
		varb["queue"] = camundaclientgo.Variable{Value: "false", Type: "boolean"}
		err := contx.Complete(processor.QueryComplete{Variables: &varb})
		if err != nil {
			log.Error("queuStatus: ", err)
			return err
		}
	} else {
		varb := contx.Task.Variables
		varb["queue"] = camundaclientgo.Variable{Value: "true", Type: "boolean"}
		err := contx.Complete(processor.QueryComplete{Variables: &varb})
		if err != nil {
			log.Error("queuStatus: ", err)
			return err
		}
	}
	log.Debug("Queue check complete.")
	return nil
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
	log.Debug("Sending email ... ")
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
	log.Debug("Email sent.")
	return d.DialAndSend(m)
}

// We use these to generate a random string
const letterBytes = "+-=abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// here we generate the random string
func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

// This is our custome theme
type SentimentTheme struct{}

func (dt *SentimentTheme) Name() string {
	return "Discourse Sentiment Theme"
}

func (dt *SentimentTheme) HTMLTemplate() string {
	// ugly hack has entered the chat ...
	contents, err := ioutil.ReadFile(ThemeTemp + ".html")
	// end ugly hack
	if err != nil { // we should really return the error here
		log.Error("sentimentTheme::HTMLTemplate: ", err)
	}
	return string(contents)
}

func (dt *SentimentTheme) PlainTextTemplate() string {
	// ugly hack has entered the chat ...
	contents, err := ioutil.ReadFile(ThemeTemp + ".txt")
	// end ugly hack
	if err != nil { // we really should return the error here
		log.Error("SentimentTheme::PlainTextTemplate: ", err)
	}
	return string(contents)
}
