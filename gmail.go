package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type Email struct {
	from    string
	to      string
	subject string
	body    []string
	date    string
	summary string
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}

	token, err := checkAndRefreshToken(tok, config, tokFile)
	if err != nil {
		panic(err)
	}

	return config.Client(context.Background(), token)
}

func getCodeParamFromURL(inputURL string) (string, error) {
	// Parse the URL
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return "", err
	}

	// Get the query parameters from the URL
	queryParams := parsedURL.Query()

	// Check if the "code" parameter is present
	codeParam := queryParams.Get("code")
	if codeParam == "" {
		return "", errors.New("code parameter not found in the URL")
	}

	return codeParam, nil
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser, authenticate Jot and then paste the "+
		"redirected url (containing 'code'): \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	code, err := getCodeParamFromURL(authCode)
	if err != nil {
		log.Fatalf("Unable to extract authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Refreshes access token if neccessary and updates in token.json
func checkAndRefreshToken(token *oauth2.Token, config *oauth2.Config, tokfile string) (*oauth2.Token, error) {
	if token.Expiry.Before(time.Now()) {
		// Token is expired, refresh it
		ctx := context.Background()          // reuse your context
		if token.Expiry.Before(time.Now()) { // expired so let's update it
			src := config.TokenSource(ctx, token)
			newToken, err := src.Token() // this actually goes and renews the tokens
			if err != nil {
				// fmt.Println("Inside error")
				tok := getTokenFromWeb(config)
				saveToken(tokfile, tok)
				return tok, nil
			}
			if newToken.AccessToken != token.AccessToken {
				saveToken(tokfile, newToken) // back to the database with new access and refresh token
				token = newToken
			}
		}
	}
	return token, nil
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// Function to retrieve the list of labels.
func getLabels(client *gmail.Service, user string) {
	r, err := client.Users.Labels.List(user).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve labels: %v", err)
	}
	if len(r.Labels) == 0 {
		fmt.Println("No labels found.")
		return
	}
	fmt.Println("Labels:")
	for _, l := range r.Labels {
		fmt.Printf("- %s\n", l.Name)
	}
}

// Get the content of the given message
func getMessageContent(msg *gmail.Message) (string, map[string]string, error) {
	var html string
	for _, part := range msg.Payload.Parts {
		if part.MimeType == "text/html" {
			data, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err != nil {
				return html, nil, err
			}
			html += string(data)
		}
	}

	headers := make(map[string]string)
	for _, header := range msg.Payload.Headers {
		switch header.Name {
		case "From",
			"To",
			"Subject",
			"Date":
			headers[header.Name] = header.Value
		}
	}
	return html, headers, nil
}

// FetchLatestMessage retrieves the latest message in the inbox of the given user.
// If no messages are found, an error is returned.
func FetchLatestMessage(client *gmail.Service, user string) (*gmail.Message, error) {
	// Retrieve the list of messages in the inbox.
	l, err := client.Users.Messages.List(user).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve messages: %v", err)
	}

	// Check if any messages were found.
	if len(l.Messages) == 0 {
		return nil, fmt.Errorf("no messages found")
	}

	// Retrieve the latest message.
	latestMessageID := l.Messages[0].Id
	msg, err := client.Users.Messages.Get(user, latestMessageID).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve message: %v", err)
	}

	return msg, nil
}

func configNotPresent(configFileName string, client *gmail.Service, user string) (uint64, error) {
	message, err := FetchLatestMessage(client, user)
	if err != nil {
		return 0, fmt.Errorf("error fetching latest message: %v", err)
	}

	// StartHistoryId is set to the historyId of latest message
	startHistoryId := message.HistoryId

	// Save startHistoryId to config file
	err = saveStartHistoryIdToConfig(startHistoryId, configFileName)
	if err != nil {
		return 0, fmt.Errorf("error saving startHistoryId to config: %v", err)
	}
	return startHistoryId, nil
}

// GetStartHistoryId retrieves the startHistoryId from the config file or fetches the latest message for starthistoryId.
func GetStartHistoryId(client *gmail.Service, user string) (uint64, error) {
	configFileName := "config.json"

	// Check if config file exists
	if _, err := os.Stat(configFileName); os.IsNotExist(err) {
		// Config file not present, fetch the latest message
		startHistoryId, err := configNotPresent(configFileName, client, user)
		return startHistoryId, err
	}

	// Config file exists, read startHistoryId from the file
	startHistoryId, err := readStartHistoryIdFromConfig(configFileName, client, user)
	if err != nil {
		return 0, fmt.Errorf("error reading startHistoryId from config: %v", err)
	}

	return startHistoryId, nil
}

// Save startHistoryId to config file.
func saveStartHistoryIdToConfig(startHistoryId uint64, fileName string) error {
	config := map[string]uint64{"startHistoryId": startHistoryId}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(fileName, data, 0644)
}

// Read startHistoryId from config file.
func readStartHistoryIdFromConfig(fileName string, client *gmail.Service, user string) (uint64, error) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return 0, err
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return 0, err
	}

	startHistoryId := uint64(config["startHistoryId"].(float64))

	if startHistoryId == 0 {
		startHistoryId, err = configNotPresent(fileName, client, user)
		if err != nil {
			return startHistoryId, err
		}
	}

	return startHistoryId, nil
}

// GetMessagesAddedinHistory retrieves the list of messages added in the history after the given history id.
// The function returns a slice of message IDs and the latest history id.
func GetMessagesAddedinHistory(history_id uint64, client *gmail.Service, user string) ([]string, uint64, error) {
	// Retrieve the history of the user.
	history, err := client.Users.History.List(user).StartHistoryId(history_id).Do()
	if err != nil {
		fmt.Println("unable to retrieve history: ", err)
		return nil, 0, fmt.Errorf("unable to retrieve history: %v", err)
	}

	// Initialize a slice to store the new message IDs.
	new_messages := []string{}

	// Update the latest history ID.
	latest_history_id := history.HistoryId

	// Iterate through the history and extract the message IDs.
	for _, hist := range history.History {
		message_added := hist.MessagesAdded
		for _, msg := range message_added {
			new_messages = append(new_messages, msg.Message.Id)
		}
	}

	return new_messages, latest_history_id, nil
}

func getAllTextFromHTML(htmlContent string) ([]string, error) {
	var textPortions []string

	// Parse HTML content
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	// Traverse the HTML tree to extract text
	var traverse func(node *html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.TextNode {
			// Append text content to the result slice
			textPortions = append(textPortions, strings.TrimSpace(node.Data))
		}

		// Recursive traversal of child nodes
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}

	// Start traversal from the root of the HTML tree
	traverse(doc)

	return textPortions, nil
}

func removeParenthesizedText(input string) string {
	re := regexp.MustCompile(`\s*\(.*?\)\s*`)
	return re.ReplaceAllString(input, "")
}

func formatDate(inputDate string) string {
	// Define a list of possible date formats
	dateFormats := []string{
		time.RFC1123Z,                    // "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC1123,                     // "Mon, 02 Jan 2006 15:04:05 MST"
		time.RFC3339,                     // "2006-01-02T15:04:05Z07:00"
		"Mon, 2 Jan 2006 15:04:05 -0700", // Custom layout to handle one-digit day
	}

	// Remove parenthesized text from the input date
	cleanedInputDate := removeParenthesizedText(inputDate)

	var parsedTime time.Time
	var err error

	// Try parsing the date with each format
	for _, layout := range dateFormats {
		parsedTime, err = time.Parse(layout, cleanedInputDate)
		if err == nil {
			break
		}
	}

	if err != nil {
		fmt.Println("Error parsing date:", err)
		return ""
	}

	// Define the layout for the output date format
	outputLayout := time.RFC3339

	// Format the parsed time to UTC and then to the desired output format
	outputDate := parsedTime.UTC().Format(outputLayout)

	return outputDate
}

func parseEmails(messages []string, client *gmail.Service, user string) ([]Email, error) {
	var newEmails []Email

	for _, message := range messages {
		// Get the message content
		fmt.Println("Getting message : ", message)
		msg, err := client.Users.Messages.Get(user, message).Do()
		if err != nil {
			fmt.Printf("Unable to retrieve %v: %v", message, err)
			continue
		}

		// Get all the text from the HTML content
		html, headers, err := getMessageContent(msg)
		if err != nil {
			log.Fatalf("Unable to get Content: %v", err)
		}

		content, err := getAllTextFromHTML(html)
		if err != nil {
			log.Fatalf("Unable to get text: %v", err)
		}
		outputDate := formatDate(headers["Date"])
		newEmails = append(newEmails, Email{headers["From"], headers["To"], headers["Subject"], content, outputDate, ""})
	}

	return newEmails, nil
}

func getEmails(emailChnl chan<- Email, wg *sync.WaitGroup) []Email {
	ctx := context.Background()
	defer wg.Done()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	user := "me"

	start_history_id, err := GetStartHistoryId(srv, user)
	if err != nil {
		log.Fatalf("Unable to retrieve startHistoryId: %v", err)
	}

	fmt.Println("Fetching history for ", start_history_id)

	new_messages, latest_history_id, err := GetMessagesAddedinHistory(start_history_id, srv, user)
	if err != nil {
		log.Fatalf("Unable to get messages: %v", err)
	}

	err = saveStartHistoryIdToConfig(latest_history_id, "config.json")
	if err != nil {
		log.Fatalf("Unable to save startHistoryId to config: %v", err)
	}

	emails, err := parseEmails(new_messages, srv, user)
	if err != nil {
		log.Fatalf("Unable to parse emails: %v", err)
	}

	fmt.Printf("You have %d new Messages", len(emails))

	for _, email := range emails {
		emailChnl <- email
	}
	close(emailChnl)
	return emails
}
