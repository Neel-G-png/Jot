package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/huggingface"
	"github.com/tmc/langchaingo/prompts"
)

type Configuration struct {
	HuggingFace_AccessToken string
	Google_AccessToken      string
	Gmail_AccessToken       string
}

func generatePrompt(email string) string {
	prompt := prompts.NewPromptTemplate(`
		[INST] Extract action items from the following Paragraph. If there are no action items, summarize the Paragraph. The final result should be presented as a JSON array of strings of action items assigned to a variable named 'ActionItems'. If no action items are present, then the array should contain a single summary string assigned to the same variable.

		The output must be in the following format: "{'ActionItems':[...]}"
		***********************************************************
		Paragraph:
		{{.Email}}?
		***********************************************************
		[/INST]`,
		[]string{"Email"},
	)
	result, err := prompt.Format(map[string]any{
		"Email": email,
	})
	if err != nil {
		fmt.Println("prompt error")
		log.Fatal(err)
	}
	return result
}

func ParseJson(jsonString string) ([]string, error) {
	type Response struct {
		ActionItems []string `json:"ActionItems"`
	}

	var response Response
	err := json.Unmarshal([]byte(jsonString), &response)
	if err != nil {
		// fmt.Println("Error parsing JSON: ", err)
		return []string{}, err
	}
	return response.ActionItems, nil
}

func cleanResult(result string) []string {
	lowerHalf := strings.SplitAfter(result, "[/INST]")
	ActionItems, _ := ParseJson(lowerHalf[1])
	return ActionItems
}

func getNewClient() *huggingface.LLM {
	clientOptions := []huggingface.Option{
		huggingface.WithToken(os.Getenv("HUGGINGFACEHUB_API_TOKEN")),
		huggingface.WithModel("mistralai/Mistral-7B-Instruct-v0.1"),
	}
	llm, err := huggingface.New(clientOptions...)
	if err != nil {
		fmt.Println("new error")
		log.Fatal(err)
	}
	return llm
}

func extractActionItems(prompt string) []string {
	llm := getNewClient()
	ctx := context.Background()
	generateOptions := []llms.CallOption{
		llms.WithModel("mistralai/Mistral-7B-Instruct-v0.1"),
		llms.WithMinLength(50),
		llms.WithMaxLength(400),
	}
	completion, err := llm.Call(ctx, prompt, generateOptions...)
	// Check for errors
	if err != nil {
		fmt.Println("call error")
		log.Fatal(err)
	}
	finalResult := cleanResult(completion)

	return finalResult
}

func process(emailChnl <-chan Email, llmChnl chan<- Email, wg *sync.WaitGroup) {
	defer wg.Done()
	for email := range emailChnl {
		emailString := "From: " + email.from + "\nTo: " + email.to + "\nSubject: " + email.subject
		for _, content := range email.body {
			emailString += "\n" + content
		}
		result := generatePrompt(emailString)
		finalResult := extractActionItems(result)
		email.summary = finalResult
		llmChnl <- email
	}
	close(llmChnl)
}

func main() {
	var wg sync.WaitGroup

	emailChnl := make(chan Email, 10)
	llmChnl := make(chan Email, 10)

	wg.Add(3)
	go getEmails(emailChnl, &wg)
	go process(emailChnl, llmChnl, &wg)
	go updateNotion(llmChnl, &wg)

	// emails := getEmails()

	for email := range llmChnl {
		fmt.Printf("\n\nFrom: %s\nTo: %s\nSubject: %s\n\n", email.from, email.to, email.subject)
		fmt.Println("Summary: ", email.summary)
	}
	wg.Wait()
	fmt.Println("All goroutines have finished execution.")
	// updateNotion(emails)
}
