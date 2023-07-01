package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/gorilla/handlers"
	"github.com/joho/godotenv"
	"golang.org/x/net/html"

	log "github.com/sirupsen/logrus"
)

var MAPSERVER_EXEC = ""

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

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

	handler := handlers.CORS(originsOk, headersOk, methodsOk)(router)

	log.Info("Listening at " + host)

	http.ListenAndServe(host, handler)
}

func handleTile(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.RawQuery
	b, err := renderTile(MAPSERVER_EXEC, queryString)

	if queryString == "" || err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Add("Content-Type", "image/jpeg")
	w.Header().Add("Content-Disposition", "inline; filename=\"tile.jpeg\"")
	w.Write(b)
}

func renderTile(mapServerExecPath, queryString string) ([]byte, error) {
	queryString = fmt.Sprintf("QUERY_STRING=%s", queryString)

	cmd := exec.Command(
		mapServerExecPath,
		queryString,
	)

	out, err := cmd.Output()
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

	response = strings.ReplaceAll(response, "Content-Type: image/jpeg", "")
	response = strings.TrimSpace(response)

	return []byte(response), nil
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

func LoadDotEnvFile(path string) {
	if err := godotenv.Load(path); err != nil {
		log.Println("Error loading env file")
	}
}

func GetMapServerPath() string {
	envVar := os.Getenv("MAPSERVER_WRAPPER_MAP_SERV_EXEC_PATH")

	if envVar == "" {
		log.Warn("MAPSERVER_WRAPPER_ALLOWED_ORIGINS environment variable is not defined. ")
		panic("MAPSERVER_WRAPPER_MAP_SERV_EXEC_PATH is no defined")
	}

	return envVar
}

func GetAllowedOrigins() []string {
	envVar := os.Getenv("MAPSERVER_WRAPPER_ALLOWED_ORIGINS")

	if envVar != "" {
		return strings.Split(envVar, ",")
	}

	log.Warn("MAPSERVER_WRAPPER_ALLOWED_ORIGINS environment variable is not defined. Accepting only local connections")

	return []string{GetLocalHost()}
}

func GetApiPort() int {
	envVar := os.Getenv("MAPSERVER_WRAPPER_PORT")

	if envVar != "" {
		port, err := strconv.Atoi(envVar)

		if err == nil {
			return port
		}

		log.Warn("Cannot parse MAPSERVER_WRAPPER environment variable. Port must be an integer.")
	}

	log.Warn("MAPSERVER_WRAPPER is not defined. Using default port 8000.")

	return 8000
}

func GetLocalHost() string {
	logLevel := log.GetLevel()

	log.SetLevel(0)
	port := GetApiPort()
	log.SetLevel(logLevel)

	return fmt.Sprintf("http://localhost:%d", port)
}

func GetRootPath() string {
	root := os.Getenv("MAPSERVER_WRAPPER_BASE_PATH")

	if root != "" && len(root) > 0 {
		if root[0] == '/' {
			return root
		}
	}

	log.Warn("MAPSERVER_WRAPPER_BASE_PATH environment variable is not defined. Default is /")

	return "/"
}

func GetCachePath() string {
	path := os.Getenv("MAPSERVER_WRAPPER_CACHE_PATH")

	if path != "" {
		return path
	}

	log.Warn("MAPSERVER_WRAPPER_CACHE_PATH environment variable is not defined." +
		"Cached tiles will be stored in ./cache folder")

	return "cache"
}
