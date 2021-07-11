//Package slack implements bot integration to slack
package slack

import (
	"html/template"
	"net/http"
	"os"

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
	botToken  = os.Getenv("BOT_TOKEN")
	oauthTmpl = template.Must(template.ParseFiles("./oauth.html"))
)

type oauthPage struct {
	Message string
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

	//Get token
	log.Infof("Returned code is: %v", code)

	if _, err := oauthConfig.Exchange(c, code); err != nil {
		log.Errorf("Error authorizing against Slack: %s", err)
		http.Error(w, "Unexpected error authorizing against Slack.", http.StatusInternalServerError)
		return
	}

	//Send to HMTL oauth success response page
	if err := oauthTmpl.Execute(w, &oauthPage{"Welcome! You can now run the slash command."}); err != nil {
		log.Errorf("Error executing oauthTmpl template: %s", err)
	}
}
