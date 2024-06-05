package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

func updateNotion(llmChnl <-chan Email, wg *sync.WaitGroup) {
	defer wg.Done()
	current_time := time.Now().UTC()
	integrationSecret := ""
	parentPageID := ""
	year, month, day := current_time.Date()
	dbName := fmt.Sprintf("%d-%02d-%02d-Database", year, month, day)

	// Create a database with todays date,
	// get the email date and check if the database corresponding to that date exists
	// if yes, add the email summary to that database
	// else, create a new database and add summary

	dbInfoList, err := readDatabaseInfo(dbName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading database info: %v\n", err)
		os.Exit(1)
	}
	dbID, dbExists := findDatabaseID(dbInfoList, dbName)

	if !dbExists {
		newDBID, err := createNotionDatabase(integrationSecret, parentPageID, dbName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating database: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Database created successfully with ID: %s\n", dbID)

		dbInfoList.Databases = append(dbInfoList.Databases, DatabaseInfo{Name: dbName, ID: newDBID})

		err = writeDatabaseInfo(dbInfoList)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing database info: %v\n", err)
			os.Exit(1)
		}
		dbID = newDBID
	}

	for email := range llmChnl {
		err = addPageToDatabase(integrationSecret, dbID, email)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n\nError adding page to database: %v\n", err)
			// os.Exit(1)
		}
	}

	fmt.Println("Page added successfully to the database")
}
