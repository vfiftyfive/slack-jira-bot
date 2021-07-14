//Package slack implements bot integration to slack
package slack

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var (
	//oauthConfig represents the oauth configuration
	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://slack.com/oauth/v2/authorize",
			TokenURL: "https://slack.com/api/oauth.v2.access",
		},
		Scopes: []string{"channels:read",
			"chat:write",
			"chat:write.customize ",
			"im:read", "im:write",
			"incoming-webhook",
			"chat:write.public",
			"reactions:write",
			"app_mentions:read"},
	}
	signingSecret = os.Getenv("SIGNING_SECRET")
	jiraURL       = "https://aviatrix.atlassian.net/rest/api/2/search"
	userName      = "nvermande@aviatrix.com"
	jiraAPIToken  = os.Getenv("JIRA_API_TOKEN")
)

type oauthPage struct {
	Message string
}

//Status represents the Jira issue status
type Status struct {
	Name string `json:"name"`
}

//Field represents the fields returned by the Jira API request
type Field struct {
	Summary string `json:"summary"`
	Status  Status `json:"status"`
}

//Issue repesents the Jira Issue
type Issue struct {
	ID     string `json:"id"`
	Link   string `json:"self"`
	Key    string `json:"key"`
	Fields Field  `json:"fields"`
}

//JiraAPIResponse is the API top-level response.
type JiraAPIResponse struct {
	Issues []Issue `json:"issues"`
}

//RichFormat holds a list of formatted blocks
type RichFormat struct {
	Blocks []Block `json:"blocks"`
}

//Text is the Slack formatted text object
type Text struct {
	Type       string `json:"type"`
	SimpleText string `json:"text"`
}

//Block is the top-object representing slack rich format payload
type Block struct {
	Type string `json:"type"`
	Text Text   `json:"text"`
}

//OauthHandler handles application install in user workspace
func OauthHandler(w http.ResponseWriter, r *http.Request) {
	c := r.Context()

	//Check for errors and get code sent by slack once user has authorized the app
	errStr := r.FormValue("error")
	if errStr != "" {
		http.Error(w, errStr, http.StatusUnauthorized)
		return
	}
	code := r.FormValue("code")
	if code == "" {
		http.Error(w, "Code is empty!!!", http.StatusBadRequest)
		log.Fatal("Error processing Oauth code")
	}

	//Retrieve access token
	log.Infof("Returned code is: %v", code)

	if _, err := oauthConfig.Exchange(c, code); err != nil {
		log.Errorf("Error authorizing against Slack: %s", err)
		http.Error(w, "Unexpected error authorizing against Slack.", http.StatusInternalServerError)
		return
	}

	//Send to HMTL oauth success response page
	oauthTmpl := template.Must(template.ParseFiles("./serverless_function_source_code/oauth.html"))
	if err := oauthTmpl.Execute(w, &oauthPage{"Welcome! You can now run the slash command."}); err != nil {
		log.Errorf("Error executing oauthTmpl template: %s", err)
	}
}

//IssueSearchHandler searches Jira issue given Mantis number and returns the issue link in Jira
func DevIssueSearchHandler(w http.ResponseWriter, r *http.Request) {
	//get headers for signature calculation
	slackTimestamp := r.Header.Get("X-Slack-Request-Timestamp")
	slackVersion := "v0:"
	slackSignature := r.Header.Get("X-Slack-Signature")

	//read body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	bodyString := string(body)

	//Compare computed signature with request signature
	slackBaseString := slackVersion + slackTimestamp + ":" + string(bodyString)
	if err != nil {
		log.Errorf("strconv.ParseInt(%s): %v", slackTimestamp, err)
	}
	h := hmac.New(sha256.New, []byte(signingSecret))
	h.Write([]byte(slackBaseString))

	sha := hex.EncodeToString(h.Sum(nil))
	sha = "v0=" + sha

	if sha != slackSignature {
		log.Errorf("Signature mismatch")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//Parse Slash command to get Mantis ID
	slashText := r.FormValue("text")
	log.Infof("Mantis ID is: %v", slashText)

	//Get redirect_url webhook to POST second message with results
	responseURL := r.FormValue("response_url")
	log.Infof("Response URL is %v", responseURL)

	//Send immediate HTTP 200
	w.WriteHeader((http.StatusOK))
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(""))

	//Use Jira API to find issue # corresponding to the Mantis (SlashText)
	JQL := "project = AVX AND Mantis[URL] = 'https://mantis.aviatrix.com/mantisbt/view.php?id=" + slashText + "'"
	jsonString := "{\"jql\": \"" + JQL + "\"," +
		"\"fields\": [" +
		"\"key\"," +
		"\"status\"," +
		"\"summary\"" +
		"]" +
		"}"

	//Create JSON string payload
	jsonData := strings.NewReader(jsonString)

	//Create new HTTP request
	req, err := http.NewRequest("POST", jiraURL, jsonData)
	if err != nil {
		log.Errorf("Error received: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	req.SetBasicAuth(userName, jiraAPIToken)

	//Use Jira API to find issue number given Mantis Id
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Errorf("Error with POST method on resource %v: %v", jiraURL, err)
	}

	//Process HTTP Response
	var j JiraAPIResponse
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&j)
	if err != nil {
		log.Errorf("Cannot decode object. Error: %v", err)
	}
	defer resp.Body.Close()

	//Return data
	if len(j.Issues) > 0 {
		returnText := Text{
			Type:       "mrkdwn",
			SimpleText: fmt.Sprintf(":raised_hands: Found Jira ID %v. Here's the hyperlink: https://aviatrix.atlassian.net/browse/%v", j.Issues[0].Key, j.Issues[0].Key),
		}

		summaryText := Text{
			Type:       "mrkdwn",
			SimpleText: fmt.Sprintf("*Issue Summary*\n%v", j.Issues[0].Fields.Summary),
		}

		statusText := Text{
			Type:       "mrkdwn",
			SimpleText: fmt.Sprintf("*Issue Status*\n%v", j.Issues[0].Fields.Status.Name),
		}

		mantisText := Text{
			Type:       "plain_text",
			SimpleText: fmt.Sprintf("Mantis %v", slashText),
		}

		mantisBlock := Block{
			Type: "header",
			Text: mantisText,
		}

		firstBlock := Block{
			Type: "section",
			Text: returnText,
		}

		secondBlock := Block{
			Type: "section",
			Text: summaryText,
		}

		thirdBlock := Block{
			Type: "section",
			Text: statusText,
		}
		s := RichFormat{[]Block{mantisBlock, firstBlock, secondBlock, thirdBlock}}

		//Create JSON string payload
		jsonResponse, _ := (json.Marshal(s))
		jsonData := strings.NewReader(string(jsonResponse))

		//Create new HTTP request
		req, err := http.NewRequest("POST", responseURL, jsonData)
		if err != nil {
			log.Errorf("Error received: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		//Post Request to webhook
		httpClient := &http.Client{
			Timeout: time.Second * 10,
		}
		if _, err = httpClient.Do(req); err != nil {
			log.Errorf("Error with POST method on resource %v: %v", jiraURL, err)
		}
	} else {
		//Create response for miss
		returnText := Text{
			Type:       "mrkdwn",
			SimpleText: fmt.Sprintf(":x: Sorry! Couldn't find any match for Mantis %v in Jira.", slashText),
		}

		firstBlock := Block{
			Type: "section",
			Text: returnText,
		}
		s := RichFormat{[]Block{firstBlock}}

		//Create JSON string payload
		jsonResponse, _ := (json.Marshal(s))
		jsonData := strings.NewReader(string(jsonResponse))

		//Create new HTTP request
		req, err := http.NewRequest("POST", responseURL, jsonData)
		if err != nil {
			log.Errorf("Error received: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		//Post Request to webhook
		httpClient := &http.Client{
			Timeout: time.Second * 10,
		}
		if _, err = httpClient.Do(req); err != nil {
			log.Errorf("Error with POST method on resource %v: %v", responseURL, err)
		}
	}
}
