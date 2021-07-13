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

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

//oauthConfig represents the oauth configuration
var (
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
			"app_me ntions:read"},
	}
	botToken     = os.Getenv("BOT_TOKEN")
	jiraURL      = "https://aviatrix.atlassian.net"
	userName     = "nvermande@aviatrix.com"
	JiraAPIToken = os.Getenv("JIRA_API_TOKEN")
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
	ID     int    `json:"id"`
	Link   string `json:"self"`
	Key    string `json:"key"`
	Fields Field  `json:"field"`
}

//JiraAPIResponse is the API top-level response.
type JiraAPIResponse struct {
	Issues []Issue `json:"issues"`
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
func IssueSearchHandler(w http.ResponseWriter, r *http.Request) {
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

	//Compare computed signature with request signature
	slackBaseString := slackVersion + slackTimestamp + ":" + string(body)
	if err != nil {
		log.Errorf("strconv.ParseInt(%s): %v", slackTimestamp, err)
	}
	h := hmac.New(sha256.New, []byte(botToken))
	h.Write([]byte(slackBaseString))

	sha := hex.EncodeToString(h.Sum(nil))
	sha = "v0=" + sha

	if sha != slackSignature {
		log.Errorf("Signature mismatch")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//Parse Slash command to get Mantis Id
	slashText := r.FormValue("text")

	//Use Jira API to find issue # corresponding to the Mantis (SlashText)
	JQL := "project = AVX AND Mantis[URL] = https:\\\\u002f\\\\u002fmantis.aviatrix.com\\\\u002fmantisbt\\\\u002fview.php\\\\u003fid\\\\u003d" + slashText
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
	req.SetBasicAuth(userName, JiraAPIToken)

	//Use Jira API to find issue number give Mantis Id
	resp, err := http.DefaultClient.Do(req)
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
	response := fmt.Sprintf("Found Jira ID: %v. Here's the hyperlink: https://aviatrix.atlassian.net/browse/%v\nSummary: %v", j.Issues[0].Key, j.Issues[0].Key, j.Issues[0].Fields.Summary)
	w.Write([]byte(response))
}
