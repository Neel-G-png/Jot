package main

import (
	"context"
	"fmt"
	"log"
	"strings"

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
	prompt := prompts.NewPromptTemplate(
		`Extract Action items from the following Paragraph into a list: 
		***********************************************************
		Paragraph : {{.Email}}?
		***********************************************************
		Action Items:`,
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

func cleanResult(result string) []string {
	actionItemsString := strings.SplitAfter(result, "Action Items:")
	actionItems := strings.SplitAfter(actionItemsString[1], "\n")
	finalResult := actionItems[1 : len(actionItems)-1]
	for i := 0; i < len(finalResult); i++ {
		finalResult[i] = strings.TrimSpace(finalResult[i])[2:]
	}
	return finalResult
}

func getNewClient() *huggingface.LLM {
	clientOptions := []huggingface.Option{
		huggingface.WithToken("XXXXXXXXXXXXX"),
		huggingface.WithModel("mistralai/Mistral-7B-v0.1"),
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
		llms.WithModel("mistralai/Mistral-7B-v0.1"),
		llms.WithMinLength(50),
		llms.WithMaxLength(400),
	}
	completion, err := llm.Call(ctx, prompt, generateOptions...)
	// Check for errors
	if err != nil {
		fmt.Println("call error")
		log.Fatal(err)
	}
	// fmt.Println(completion, "\n\n########################")
	finalResult := cleanResult(completion)

	return finalResult //[]string{completion}
}

func main() {
	// email := "I trust this message finds you well. As we navigate the demands of our current projects. First and foremost, we have a client presentation scheduled for 2:00 PM, and I need you to finalize the presentation deck by incorporating all relevant data, charts, and key insights. Additionally, we need your expertise in reviewing the budget proposal for Project X, providing detailed feedback, and coordinating with the finance team on any necessary adjustments. Lastly, please compile individual progress reports from team members for our weekly meeting tomorrow, summarizing key achievements, challenges, and upcoming goals. Kindly submit the compiled report to me by 5:00 PM today for review. Your dedication to completing these tasks promptly is instrumental to our success. Feel free to reach out if you have any questions or foresee any challenges."
	emails := getEmails()
	// Make each email into a string.
	var actionItems [][]string
	for _, email := range emails {
		emailString := "From: " + email.from + "\nTo: " + email.to + "\nSubject: " + email.subject

		for _, content := range email.body {
			emailString += "\n" + content
		}
		result := generatePrompt(emailString)
		finalResult := extractActionItems(result)
		actionItems = append(actionItems, finalResult)
	}
	fmt.Println(actionItems)
	// updateNotion(actionItems)
	// fmt.Println("This is the final result : ", finalResult)
}
