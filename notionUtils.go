package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"net/http"
	"os"
)

const (
	notionAPIBaseURL = "https://api.notion.com/v1/"
	notionAPIVersion = "2021-05-13"
	databaseFileName = "databases.json"
)

type NotionDatabase struct {
	Parent     Parent              `json:"parent"`
	Title      []RichText          `json:"title"`
	Properties map[string]Property `json:"properties"`
}

type Parent struct {
	Type       string `json:"type"`
	PageID     string `json:"page_id"`
	DatabaseID string `json:"database_id"`
}

type RichText struct {
	Type        string      `json:"type"`
	Text        TextContent `json:"text"`
	Annotations Annotations `json:"annotations"`
	PlainText   string      `json:"plain_text"`
	Href        string      `json:"href"`
}

type TextContent struct {
	Content string `json:"content"`
}

type Annotations struct {
	Bold          bool `json:"bold"`
	Italic        bool `json:"italic"`
	Strikethrough bool `json:"strikethrough"`
	Underline     bool `json:"underline"`
	Code          bool `json:"code"`
}

type Property struct {
	Type     string    `json:"type"`
	Title    *struct{} `json:"title,omitempty"`
	RichText *struct{} `json:"rich_text,omitempty"`
	Date     *struct{} `json:"date,omitempty"`
}

type NotionDatabaseResponse struct {
	ID string `json:"id"`
}

type PageProperties struct {
	Title    []RichText     `json:"title,omitempty"`
	RichText []RichText     `json:"rich_text,omitempty"`
	Date     *Date          `json:"date,omitempty"`
	Checkbox *CheckboxValue `json:"checkbox,omitempty"`
}

type Date struct {
	Start string      `json:"start"`
	End   interface{} `json:"end,omitempty"`
}

type CheckboxValue struct {
	Name  string `json:"name"`
	Value bool   `json:"value"`
}

type Page struct {
	Parent     Parent                    `json:"parent"`
	Properties map[string]PageProperties `json:"properties"`
}

type DatabaseInfo struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type DatabaseInfoList struct {
	Databases []DatabaseInfo `json:"databases"`
}

func findDatabaseID(dbInfoList DatabaseInfoList, dbName string) (string, bool) {
	for _, db := range dbInfoList.Databases {
		if db.Name == dbName {
			return db.ID, true
		}
	}
	return "", false
}

func readDatabaseInfo(dbName string) (DatabaseInfoList, error) {
	var dbInfoList DatabaseInfoList

	file, err := os.ReadFile(databaseFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return dbInfoList, nil
		}
		return dbInfoList, err
	}

	err = json.Unmarshal(file, &dbInfoList)
	if err != nil {
		return dbInfoList, err
	}
	return dbInfoList, nil
}

func writeDatabaseInfo(dbInfoList DatabaseInfoList) error {
	file, err := json.MarshalIndent(dbInfoList, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(databaseFileName, file, 0644)
	if err != nil {
		return err
	}

	return nil
}

func createNotionDatabase(integrationSecret, parentPageID, dbName string) (string, error) {
	url := notionAPIBaseURL + "databases"
	client := &http.Client{}

	database := NotionDatabase{
		Parent: Parent{
			Type:   "page_id",
			PageID: parentPageID,
		},
		Title: []RichText{
			{
				Type: "text",
				Text: TextContent{
					Content: dbName,
				},
				Annotations: Annotations{
					Bold: true,
				},
				PlainText: dbName,
			},
		},
		Properties: map[string]Property{
			"Email From": {
				Type:  "title",
				Title: &struct{}{},
			},
			"Date": {
				Type: "date",
				Date: &struct{}{},
			},
			"Subject": {
				Type:     "rich_text",
				RichText: &struct{}{},
			},
			"Summary": {
				Type:     "rich_text",
				RichText: &struct{}{},
			},
		},
	}

	jsonData, err := json.Marshal(database)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+integrationSecret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", notionAPIVersion)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create database: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var notionResp NotionDatabaseResponse
	err = json.Unmarshal(body, &notionResp)
	if err != nil {
		return "", err
	}

	return notionResp.ID, nil
}

func addPageToDatabase(integrationSecret, databaseID string, email Email) error {
	url := notionAPIBaseURL + "pages"
	client := &http.Client{}

	page := Page{
		Parent: Parent{
			Type:       "database_id",
			DatabaseID: databaseID,
		},
		Properties: map[string]PageProperties{
			"Email From": {
				Title: []RichText{
					{
						Type: "text",
						Text: TextContent{
							Content: email.from,
						},
					},
				},
			},
			"Date": {
				Date: &Date{
					Start: email.date,
				},
			},
			"Summary": {
				RichText: []RichText{
					{
						Type: "text",
						Text: TextContent{
							Content: email.summary,
						},
						Annotations: Annotations{
							Bold:          false,
							Italic:        false,
							Strikethrough: false,
							Underline:     false,
							Code:          false,
						},
						PlainText: email.summary,
					},
				},
			},
			"Subject": {
				RichText: []RichText{
					{
						Type: "text",
						Text: TextContent{
							Content: email.subject,
						},
						Annotations: Annotations{
							Bold:          false,
							Italic:        false,
							Strikethrough: false,
							Underline:     false,
							Code:          false,
						},
						PlainText: email.subject,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(page)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+integrationSecret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", notionAPIVersion)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add page: %s, response: %s", resp.Status, string(body))
	}

	return nil
}
