package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/huggingface"
	"github.com/tmc/langchaingo/prompts"
)

func main() {
	prompt := prompts.NewPromptTemplate(
		`Extract Action items from the following paragraph into a list : 
		***********************************************************
		Paragraph : {{.text}}?
		***********************************************************`,
		[]string{"text"},
	)

	result, err := prompt.Format(map[string]any{
		"text": "I trust this message finds you well. As we navigate the demands of our current projects. First and foremost, we have a client presentation scheduled for 2:00 PM, and I need you to finalize the presentation deck by incorporating all relevant data, charts, and key insights. Additionally, we need your expertise in reviewing the budget proposal for Project X, providing detailed feedback, and coordinating with the finance team on any necessary adjustments. Lastly, please compile individual progress reports from team members for our weekly meeting tomorrow, summarizing key achievements, challenges, and upcoming goals. Kindly submit the compiled report to me by 5:00 PM today for review. Your dedication to completing these tasks promptly is instrumental to our success. Feel free to reach out if you have any questions or foresee any challenges.",
	})
	if err != nil {
		log.Fatal(err)
	}

	clientOptions := []huggingface.Option{
		huggingface.WithModel("mistralai/Mistral-7B-v0.1"),
	}
	llm, err := huggingface.New(clientOptions...)

	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	generateOptions := []llms.CallOption{
		llms.WithModel("Falconsai/text_summarization"),
		llms.WithMinLength(50),
		llms.WithMaxLength(400),
	}
	completion, err := llm.Call(ctx, result, generateOptions...)
	// Check for errors
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(completion)
}
