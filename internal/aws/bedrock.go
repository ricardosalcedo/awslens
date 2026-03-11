package aws

import (
	"context"
	"time"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type bedrockRequest struct {
	AnthropicVersion string           `json:"anthropic_version"`
	MaxTokens        int              `json:"max_tokens"`
	Messages         []bedrockMessage `json:"messages"`
	System           string           `json:"system,omitempty"`
}

type bedrockMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type bedrockResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (c *Client) GetInsight(ctx context.Context, resourceSummary string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	body, _ := json.Marshal(bedrockRequest{
		AnthropicVersion: "bedrock-2023-05-31",
		MaxTokens:        400,
		System: `You are an expert AWS operations advisor. Given a resource config and its real operational data (metrics, errors, dependencies), provide a structured analysis:

WHAT IT DOES: One sentence describing the resource's purpose based on its name, config, and connections.
HEALTH: Current health based on metrics and errors. Flag anything abnormal.
COST: Is this resource right-sized? Any waste? Suggest savings if applicable.
ACTION ITEMS: 1-3 specific, actionable recommendations ranked by impact.

Be direct. Use the actual numbers from the metrics. No generic advice — only insights specific to this resource's data. Under 200 words.`,
		Messages: []bedrockMessage{
			{Role: "user", Content: resourceSummary},
		},
	})

	svc := bedrockruntime.NewFromConfig(c.Config)
	out, err := svc.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     strPtr("anthropic.claude-3-haiku-20240307-v1:0"),
		ContentType: strPtr("application/json"),
		Body:        body,
	})
	if err != nil {
		return "", fmt.Errorf("bedrock: %w", err)
	}

	var resp bedrockResponse
	if err := json.Unmarshal(out.Body, &resp); err != nil {
		return "", fmt.Errorf("bedrock: bad response: %w", err)
	}
	if len(resp.Content) == 0 {
		return "", fmt.Errorf("bedrock: empty response")
	}
	return resp.Content[0].Text, nil
}

func strPtr(s string) *string { return &s }
