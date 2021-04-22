package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	//"net/http/httputil"

	recaptcha "cloud.google.com/go/recaptchaenterprise/apiv1"
	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/mux"
	"golang.org/x/net/http2"
	"google.golang.org/api/option"
	recaptchapb "google.golang.org/genproto/googleapis/cloud/recaptchaenterprise/v1"
)

var (
	client *recaptcha.Client
)

const (
	saFile          = "/path/to/recaptcha-svc.json"
	checkKey        = "6LenwLMaAAAAAJiGCftfhT2Iv-redacted"
	scoringKey      = "6Lefk7QaAAAAAJAtX1S7W_redacted"
	assessmentName  = "yourassessmentname"
	parentProject   = "projects/yourproject"
	recaptchaAction = "homepage"
)

func gethandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, fmt.Sprintf("ok"))
}

func posthandler(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()

	var key string
	token := r.FormValue("token")
	apiType := r.FormValue("type")
	if apiType == "score" {
		key = scoringKey
	} else if apiType == "check" {
		key = checkKey
	} else {
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
	fmt.Printf("TOKEN :%s\n", token)
	ctx := r.Context()
	event := &recaptchapb.Event{
		ExpectedAction: recaptchaAction,
		Token:          token,
		SiteKey:        key,
	}

	assessment := &recaptchapb.Assessment{
		Event: event,
	}

	request := &recaptchapb.CreateAssessmentRequest{
		Assessment: assessment,
		Parent:     parentProject,
	}

	response, err := client.CreateAssessment(
		ctx,
		request)

	if err != nil {
		fmt.Printf("%v\n", err.Error())
		return
	}

	var resp string
	if response.TokenProperties.Valid == false {
		resp = fmt.Sprintf("The CreateAssessment() call failed because the token"+
			" was invalid for the following reasons: %v\n",
			response.TokenProperties.InvalidReason)
	} else {
		if response.Event.ExpectedAction == recaptchaAction {
			resp = fmt.Sprintf("The reCAPTCHA score for this token is:  %v\n",
				response.RiskAnalysis.Score)
			m := jsonpb.Marshaler{}
			result, err := m.MarshalToString(response)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			var prettyJSON bytes.Buffer
			err = json.Indent(&prettyJSON, []byte(result), "", "\t")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			resp = prettyJSON.String()

			fmt.Printf("Response \n%s\n", response)
		} else {
			resp = fmt.Sprintf("The action attribute in your reCAPTCHA tag does" +
				"not match the action you are expecting to score\n")
		}
	}

	fmt.Fprint(w, resp)
}

func main() {

	ctx := context.Background()
	var err error
	client, err = recaptcha.NewClient(ctx, option.WithServiceAccountFile(saFile))
	if err != nil {
		fmt.Printf("Error creating reCAPTCHA client\n")
	}

	router := mux.NewRouter()
	router.Methods(http.MethodGet).Path("/").HandlerFunc(gethandler)
	router.Methods(http.MethodPost).Path("/verifyIdToken").HandlerFunc(posthandler)
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	var server *http.Server
	server = &http.Server{
		Addr:    ":8081",
		Handler: router,
	}

	http2.ConfigureServer(server, &http2.Server{})
	fmt.Println("Starting Server..")
	err = server.ListenAndServeTLS("certs/local.crt", "certs/local.key")
	fmt.Printf("Unable to start Server %v", err)

}
