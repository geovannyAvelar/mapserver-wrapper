package main

import (
	"crypto/md5"
	"encoding/hex"
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

	envFilePath := ".env"
	args := os.Args[1:]

	if len(args) > 0 {
		envFilePath = args[0]
	}

	LoadDotEnvFile(envFilePath)

	MAPSERVER_EXEC = GetMapServerPath()

	router := chi.NewRouter()

	router.Route(GetRootPath(), func(r chi.Router) {
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

	if queryString == "" {
		http.Error(w, "Query parameters cannot be empty", http.StatusBadRequest)
		return
	}

	b, err := getTileFromDisk(queryString)

	if err == nil {
		w.Header().Add("Content-Type", "image/jpeg")
		w.Header().Add("Content-Disposition", "inline; filename=\"tile.jpeg\"")
		w.Write(b)
		return
	}

	b, imgType, err := renderTile(MAPSERVER_EXEC, queryString)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	go saveTile(queryString, b)

	w.Header().Add("Content-Type", fmt.Sprintf("image/%s", imgType))
	w.Header().Add("Content-Disposition", fmt.Sprintf("inline; filename=\"image.%s\"", imgType))
	w.Write(b)
}

func renderTile(mapServerExecPath, queryString string) ([]byte, string, error) {
	queryString = fmt.Sprintf("QUERY_STRING=%s", queryString)

	cmd := exec.Command(
		mapServerExecPath,
		queryString,
	)

	out, err := cmd.Output()
	if err != nil {
		msg := fmt.Sprintf("Failed to render tile: %v", err)
		log.Error(msg)
		return nil, "", errors.New(msg)
	}

	response := string(out)
	if strings.Contains(response, "MapServer Message") {
		message, err := parseMapServerErrorMessage(response)

		if err != nil {
			return nil, "", errors.New("cannot generate tile")
		}

		log.Errorf("cannot generate tile. Cause: %s", message)

		return nil, "", fmt.Errorf("cannot generate tile. %s", message)
	}

	imgType := ""

	if strings.Contains(response, "image/jpeg") {
		imgType = "jpeg"
	}

	if strings.Contains(response, "image/png") {
		imgType = "png"
	}

	contentType := fmt.Sprintf("Content-Type: image/%s", imgType)

	response = strings.ReplaceAll(response, contentType, "")
	response = strings.TrimSpace(response)

	return []byte(response), imgType, nil
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

func saveTile(queryString string, bytes []byte) error {
	filepath := GetCachePath() + string(os.PathSeparator) + createMd5Hash(queryString)

	if _, err := os.Stat(filepath); err == nil {
		return nil
	}

	err := os.WriteFile(filepath, bytes, 0644)

	if err != nil {
		log.Warnf("cannot create tile file. Cause: %s", err)
		return fmt.Errorf("cannot create tile file. Cause: %w", err)
	}

	return nil
}

func getTileFromDisk(queryString string) ([]byte, error) {
	filepath := GetCachePath() + string(os.PathSeparator) + createMd5Hash(queryString)

	if _, err := os.Stat(filepath); err != nil {
		return nil, errors.New("tile is not cached")
	}

	bytes, err := os.ReadFile(filepath)

	if err != nil {
		return nil, fmt.Errorf("cannot read tile from disk. Cause: %w", err)
	}

	return bytes, nil
}

func createMd5Hash(text string) string {
	hasher := md5.New()
	_, err := io.WriteString(hasher, text)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func LoadDotEnvFile(path string) {
	if err := godotenv.Load(path); err != nil {
		msg := "Cannot load '%s' environment file. " +
			"Looking for environment variables in the system environment"
		log.Warnf(msg, path)
	}
}

func GetMapServerPath() string {
	envVar := os.Getenv("MAPSERVER_WRAPPER_MAP_SERV_EXEC_PATH")

	if envVar == "" {
		fmt.Println("MAPSERVER_WRAPPER_ALLOWED_ORIGINS environment variable is not defined. Aborted.")
		os.Exit(1)
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
