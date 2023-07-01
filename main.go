package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/go-chi/chi"
	"github.com/gorilla/handlers"
	"golang.org/x/net/html"

	log "github.com/sirupsen/logrus"
)

var MAPSERVER_EXEC = ""

func main() {
	LoadDotEnvFile(".env")

	MAPSERVER_EXEC = GetMapServerPath()

	router := chi.NewRouter()

	router.Route("/", func(r chi.Router) {
		r.Get("/", handleTile)
	})

	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With"})
	originsOk := handlers.AllowedOrigins(GetAllowedOrigins())
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	port := GetApiPort()
	host := fmt.Sprintf(":%d", port)
	log.Info("Listening at " + host + GetLocalHost())

	handler := handlers.CORS(originsOk, headersOk, methodsOk)(router)

	http.ListenAndServe(host, handler)
}

func handleTile(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.RawQuery
	b, err := renderTile(MAPSERVER_EXEC, queryString)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Add("Content-Type", "image/jpeg")
	w.Write(b)
}

func renderTile(mapServerExecPath, queryString string) ([]byte, error) {
	queryString = fmt.Sprintf("QUERY_STRING=%s", queryString)

	cmd := exec.Command(
		mapServerExecPath,
		queryString,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := fmt.Sprintf("Failed to render tile: %v", err)
		log.Error(msg)
		return nil, errors.New(msg)
	}

	response := string(out)
	if strings.Contains(response, "MapServer Message") {
		message, err := parseMapServerErrorMessage(response)

		if err != nil {
			return nil, errors.New("cannot generate tile")
		}

		return nil, fmt.Errorf("cannot generate tile. %s", message)
	}

	return out, nil
}

func parseMapServerErrorMessage(htmlResponse string) (string, error) {
	tokenizer := html.NewTokenizer(strings.NewReader(htmlResponse))

	for {
		tt := tokenizer.Next()

		if tt == html.ErrorToken {
			if tokenizer.Err() == io.EOF {
				break
			}

			continue
		}

		tag, _ := tokenizer.TagName()
		if strings.Contains(string(tag), "body") {
			tokenizer.Next()
			errorMsg := string(tokenizer.Text())
			return errorMsg, nil
		}
	}

	return "", errors.New("cannot parse MapServer error message")
}
