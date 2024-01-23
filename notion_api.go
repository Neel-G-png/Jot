package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// Notion API URL constants
const (
	notionVersion = "2022-06-28"
	baseAPIURL    = "https://api.notion.com/v1"
)

func createPage(parentID, pageTitle, content, token string) (string, error) {
	data := map[string]interface{}{
		"parent": map[string]interface{}{
			"database_id": parentID,
		},
		"properties": map[string]interface{}{
			"Name": map[string]interface{}{
				"title": []map[string]interface{}{
					{
						"text": map[string]interface{}{
							"content": pageTitle,
						},
					},
				},
			},
			"Status": map[string]interface{}{
				"select": map[string]interface{}{
					"name": "To-Do", // Ensure this matches the option in your Notion database
				},
			},
		},
		"children": []map[string]interface{}{
			{
				"object": "block",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{
							"type": "text",
							"text": map[string]interface{}{
								"content": content,
							},
						},
					},
				},
			},
		},
	}

	response, err := makeNotionRequest("POST", "/pages", data, token)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed: %s", string(responseBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", err
	}

	pageID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get the new page ID")
	}

	return pageID, nil
}

// makeNotionRequest handles making HTTP requests to the Notion API
func makeNotionRequest(method, path string, data interface{}, token string) (*http.Response, error) {
	var requestBody io.Reader

	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewBuffer(jsonData)
	}

	request, err := http.NewRequest(method, baseAPIURL+path, requestBody)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", "Bearer "+token)
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Notion-Version", notionVersion)

	client := &http.Client{}
	return client.Do(request)
}

func UpdatePageTitle(pageID, newTitle, token string) error {
	// Define the request payload
	updateData := map[string]interface{}{
		"properties": map[string]interface{}{
			"title": []map[string]interface{}{
				{
					"text": map[string]interface{}{
						"content": newTitle,
					},
				},
			},
		},
	}

	response, err := makeNotionRequest("PATCH", "/pages/"+pageID, updateData, token)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %s: %s", response.Status, string(responseBody))
	}

	fmt.Println("Response Status:", response.Status)
	fmt.Println("Response Body:", string(responseBody))

	return nil
}

func updateNotion(emails [][]string) {
	token := "secret_Jgnpej2cxzLpnP1tVJBOVkMO8kidJVaYVH0H16uHoBV"
	parentID := "783e85fec3544e6684487ce244769b83" // Replace with your Notion database ID
	newTitle := "Action Items from Email"
	emailContent := "1. [Marketing Team] Finalize new campaign strategy by next Friday.\n2. [Tech Team] Complete software update testing by Wednesday.\n3. Schedule a client meeting to discuss project progress.\n4. Review and update project documentation by the end of this week."

	// Create a new page with title and content
	newPageID, err := createPage(parentID, newTitle, emailContent, token)
	if err != nil {
		fmt.Println("Error creating page:", err)
		return
	}
	fmt.Println("New page created successfully with ID:", newPageID)
}
