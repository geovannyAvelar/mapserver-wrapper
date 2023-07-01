package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

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
