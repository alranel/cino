package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alranel/cino/lib"
	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
)

func StartWebService() {
	fmt.Printf("Starting the web service on %s\n", Config.WS.Bind)

	router := mux.NewRouter()
	router.HandleFunc("/github-hook", githubHookEndpoint).Methods("POST")

	srv := &http.Server{
		Handler:      router,
		Addr:         Config.WS.Bind,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func githubHookEndpoint(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, []byte(Config.GitHub.Secret))
	if err != nil {
		log.Printf("error validating request body: err=%s\n", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	defer r.Body.Close()

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Printf("could not parse webhook: err=%s\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch e := event.(type) {
	case *github.CheckSuiteEvent:
		if *e.Action == "requested" || *e.Action == "rerequested" {
			db := lib.ConnectDB(Config.DB)

			_, err := db.Exec(`INSERT INTO check_suites 
				(github_id, github_installation_id, repo_name, repo_owner, repo_clone_url, commit_ref) 
				VALUES ($1, $2, $3, $4, $5, $6)`,
				e.GetCheckSuite().GetID(),
				e.GetInstallation().GetID(),
				e.GetRepo().GetName(),
				e.GetRepo().GetOwner().GetLogin(),
				e.GetRepo().GetCloneURL(),
				e.GetCheckSuite().GetHeadSHA(),
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	default:
		log.Printf("unknown event type %s\n", github.WebHookType(r))
		return
	}

	w.WriteHeader(200)
}
