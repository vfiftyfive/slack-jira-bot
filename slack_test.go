package slack

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestJiraAPI(t *testing.T) {
	MantisID := "12477"
	JQL := "project = AVX AND Mantis[URL] = https:\\\\u002f\\\\u002fmantis.aviatrix.com\\\\u002fmantisbt\\\\u002fview.php\\\\u003fid\\\\u003d" + MantisID
	jsonString := "{\"jql\": \"" + JQL + "\"," +
		"\"fields\": [" +
		"\"key\"," +
		"\"status\"," +
		"\"summary\"" +
		"]" +
		"}"
	fmt.Println(jsonString)

	//Create JSON string payload
	jsonData := strings.NewReader(jsonString)

	//Create new HTTP request
	req, err := http.NewRequest("POST", jiraURL, jsonData)
	if err != nil {
		t.Errorf("Error received: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(userName, jiraAPIToken)

	//Perform HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("Error with POST method on resource %v: %v", jiraURL, err)
	}
	defer resp.Body.Close()

	//Process HTTP Response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading body response: %v", err)
	}
	fmt.Printf("Body: %v\nHTTP response: %v\n", string(body), resp.StatusCode)
}

func TestSlashHandler(t *testing.T) {
	URLData := url.Values{}
	URLData.Set("text", "12477")

	//Create mock HTTP POST request
	req, err := http.NewRequest("POST", "https://us-central1-nv-avtx-compute.cloudfunctions.net/IssueSearchHandler", strings.NewReader(URLData.Encode()))
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	//Create ResponseRecorder
	rec := httptest.NewRecorder()
	handler := http.HandlerFunc(IssueSearchHandler)
	handler.ServeHTTP(rec, req)
	if status := rec.Code; status != http.StatusOK {
		t.Errorf("Handler returned unexpected status %v. Wanted %v", status, http.StatusOK)
	}

	t.Logf("Body content: %v", rec.Body.String())
}

func TestJiraCloud(t *testing.T) {
	JQL := "project = AVX AND Mantis[URL] = 'https://mantis.aviatrix.com/mantisbt/view.php?id=12477'"
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
		t.Errorf("Error received: %v", err)
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
		t.Errorf("Error with GET method on resource %v: %v", jiraURL, err)
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
	log.Infof("\U0001f64C Found Jira ID: %v. Here's the hyperlink: https://aviatrix.atlassian.net/browse/%v\nSummary: %v", j.Issues[0].Key, j.Issues[0].Key, j.Issues[0].Fields.Summary)

}
