package slack

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestJiraAPI(t *testing.T) {
	jiraURL := "https://aviatrix.atlassian.net/rest/api/2/search"
	userName := "nvermande@aviatrix.com"
	JiraApiToken := "xQtACc0gt7XpzDWCF8js0783"
	MantisId := "12477"
	// JQL := "project = AVX AND Mantis[URL Field] = https://mantis.aviatrix.com/mantisbt/view.php?id=" + MantisId
	JQL := "project = AVX AND Mantis[URL] = https:\\\\u002f\\\\u002fmantis.aviatrix.com\\\\u002fmantisbt\\\\u002fview.php\\\\u003fid\\\\u003d" + MantisId
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
	req.SetBasicAuth(userName, JiraApiToken)

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
