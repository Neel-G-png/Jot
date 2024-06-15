package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

type Config struct {
	IntegrationSecret string `json:"integrationSecret"`
	ParentPageID      string `json:"parentPageID"`
}

func getNotionCreds() Config {
	jsonFile, err := os.Open("notionCred.json")
	if err != nil {
		log.Fatalf("Failed to open JSON file: %s", err)
	}
	defer jsonFile.Close()

	// Read the file contents
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		log.Fatalf("Failed to read JSON file: %s", err)
	}

	// Unmarshal the JSON data into the struct
	var config Config
	if err := json.Unmarshal(byteValue, &config); err != nil {
		log.Fatalf("Failed to unmarshal JSON data: %s", err)
	}
	return config
}

func updateNotion(llmChnl <-chan Email, wg *sync.WaitGroup) {
	defer wg.Done()
	// current_time := time.Now().UTC()
	config := getNotionCreds()
	integrationSecret := config.IntegrationSecret
	parentPageID := config.ParentPageID

	// year, month, day := current_time.Date()
	// dbName := fmt.Sprintf("%d-%02d-%02d-Database", year, month, day)

	for email := range llmChnl {
		currEmailDate := strings.Split(email.date, "T")[0]
		// fmt.Println(currEmailDate)
		currEmailDbName := fmt.Sprintf("%s-Database", currEmailDate)

		// Create a database with todays date,
		// get the email date and check if the database corresponding to that date exists
		// if yes, add the email summary to that database
		// else, create a new database and add summary

		dbInfoList, err := readDatabaseInfo(currEmailDbName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading database info: %v\n", err)
			os.Exit(1)
		}
		dbID, dbExists := findDatabaseID(dbInfoList, currEmailDbName)

		if !dbExists {
			newDBID, err := createNotionDatabase(integrationSecret, parentPageID, currEmailDbName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating database: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Database created successfully with ID: %s\n", dbID)

			dbInfoList.Databases = append(dbInfoList.Databases, DatabaseInfo{Name: currEmailDbName, ID: newDBID})

			err = writeDatabaseInfo(dbInfoList)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing database info: %v\n", err)
				os.Exit(1)
			}
			dbID = newDBID
		}

		err = addPageToDatabase(integrationSecret, dbID, email)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n\nError adding page to database: %v\n", err)
			// os.Exit(1)
		}
	}

	fmt.Println("Page added successfully to the database")
}
