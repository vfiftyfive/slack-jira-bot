package slack

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestJiraAPI(t *testing.T) {
	jiraURL := "https://aviatrix.atlassian.net/rest/api/2/search"
	userName := "nvermande@aviatrix.com"
	JiraAPIToken := "xQtACc0gt7XpzDWCF8js0783"
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
	req.SetBasicAuth(userName, JiraAPIToken)

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
	URLData.Set("text", "94070")

	//Create mock HTTP POST request
	req, err := http.NewRequest("POST", "https://us-central1-nv-avtx-compute.cloudfunctions.net/IssueSearchHandler", strings.NewReader(URLData.Encode()))
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(URLData.Encode())))

	//Create ResponseRecorder
	rec := httptest.NewRecorder()
	handler := http.HandlerFunc(IssueSearchHandler)
	handler.ServeHTTP(rec, req)
	if status := rec.Code; status != http.StatusOK {
		t.Errorf("Handler returner unexpected status %v. Wanted %v", status, http.StatusOK)
	}

	t.Logf("Body content: %v", rec.Body.String())
}
