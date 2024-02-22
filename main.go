package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"m/models"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	CONFIG_FILE_NAME = "config.json"
	LOG_FILE_NAME    = "log.json"
)

var appLogStructure *models.LogStructure = &models.LogStructure{
	Logs: []string{},
}

func main() {

	config, err := processAppConfig()
	if err != nil {
		log.Fatalln("Failed to process provided JSON file containing the list of URLs")
	}

	httpClient := &http.Client{}
	wg := sync.WaitGroup{}
	var next string
	mu := &sync.Mutex{}

	urlConfig := processTargetFile(config)

	for i := 0; i < config.NumGoRoutines; i++ {
		wg.Add(1)

		go func(client *http.Client) {
			defer wg.Done()

			for {
				mu.Lock() // Lock before modifying shared memory

				// Pop an item off the front of the list
				next, urlConfig.URLs = urlConfig.URLs[0], urlConfig.URLs[1:]
				if len(strings.TrimSpace(next)) > 0 {
					checkURL(next, client)
				}

				// Push the processed item back on the list so we
				// continuously monitor all URLs.
				urlConfig.URLs = append(urlConfig.URLs, next)
				mu.Unlock()
			}
		}(httpClient)
	}
	wg.Wait() // block forever
}

func checkURL(ul string, client *http.Client) {

	parsedUrl, err := url.Parse(ul)
	if err != nil {
		return
	}

	resp, err := client.Do(&http.Request{
		Method: http.MethodGet,
		URL:    parsedUrl,
	})
	if err != nil {
		return
	}
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)

	var upOrDown string = "DOWN"
	okStatusCode := resp.StatusCode >= http.StatusOK && resp.StatusCode <= http.StatusIMUsed
	if okStatusCode {
		upOrDown = "UP"
	}

	logEntry := fmt.Sprintf("HTTP Status Code: %v :: Status: %v :: Timestamp UTC: %v :: URL: %v", resp.StatusCode, upOrDown, time.Now().UTC().String(), ul)
	if err != nil {
		logEntry = fmt.Sprintf("HTTP Status Code: %v :: Status: %v :: Timestamp UTC: %v :: URL: %v :: Error: %v", resp.StatusCode, upOrDown, time.Now().UTC().String(), ul, err.Error())
	}

	log.Println(logEntry)
	appLogStructure.Logs = append(appLogStructure.Logs, logEntry)
	writeLog()
}

func writeLog() {
	logEntryJSON, err := json.MarshalIndent(appLogStructure, "", "    ")
	if err != nil {
		return
	}

	logFile, err := os.OpenFile(LOG_FILE_NAME, os.O_RDWR, os.ModePerm)
	if err != nil {
		return
	}
	defer logFile.Close()
	logFile.Write([]byte(logEntryJSON))
}

func processAppConfig() (*models.AppConfiguration, error) {

	jsonfile, err := os.Open(CONFIG_FILE_NAME)
	if err != nil {
		return nil, err
	}
	defer jsonfile.Close()

	bytes, err := io.ReadAll(jsonfile)
	if err != nil {
		return nil, err
	}

	// Decode the file contents into a structured object format
	var config *models.AppConfiguration
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func processTargetFile(appConfig *models.AppConfiguration) *models.URLFileConfiguration {
	// Since we're intentionally stopping the application when there's a critical error in the scope
	// of this function we don't need to return an error from here at this time.

	// Reset log file first
	if err := os.Truncate(LOG_FILE_NAME, 0); err != nil {
		log.Fatalf("Failed to reset log file: %v", err)
	}

	// Open the file and convert the URLs to a Go slice
	jsonfile, err := os.Open(appConfig.TargetFile)
	if err != nil {
		log.Fatalln("Failed to open URL file")
	}
	defer jsonfile.Close()

	// Read in URL file
	bytes, err := io.ReadAll(jsonfile)
	if err != nil {
		log.Fatalln("Failed to read URL file")
	}

	var urlFileConfig *models.URLFileConfiguration
	err = json.Unmarshal(bytes, &urlFileConfig)
	if err != nil {
		log.Fatalln("Failed to deserialize URL file")
	}
	return urlFileConfig
}
