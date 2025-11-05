package handlers

import (
	"context"
	"encoding/base64"
	"treasureHunt/config"

	"github.com/gofiber/fiber/v2"
	openai "github.com/sashabaranov/go-openai"
)

type ImageHandler struct {
	OpenAIClient *openai.Client
}

func NewImageHandler() *ImageHandler {
	client := openai.NewClient(config.AppConfig.OpenAIAPIKey)
	return &ImageHandler{
		OpenAIClient: client,
	}
}

type ImageAnalysisRequest struct {
	ImageURL    string `json:"imageUrl"`    // URL of the image
	ImageBase64 string `json:"imageBase64"` // Base64-encoded image (alternative to URL)
	Prompt      string `json:"prompt"`      // Question to ask about the image
}

type ImageAnalysisResponse struct {
	Result bool `json:"result"` // true/false answer
}

// GetImageContents analyzes an image using OpenAI Vision API
func (h *ImageHandler) GetImageContents(c *fiber.Ctx) error {
	var req ImageAnalysisRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate input
	if req.Prompt == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Prompt is required",
		})
	}

	if req.ImageURL == "" && req.ImageBase64 == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Either imageUrl or imageBase64 is required",
		})
	}

	// Prepare the image content
	var imageContent openai.ChatMessagePart
	if req.ImageURL != "" {
		imageContent = openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL: req.ImageURL,
			},
		}
	} else {
		// Decode base64 if needed (OpenAI expects data URI format)
		dataURI := req.ImageBase64
		if _, err := base64.StdEncoding.DecodeString(req.ImageBase64); err == nil {
			// If it's plain base64, add the data URI prefix
			dataURI = "data:image/jpeg;base64," + req.ImageBase64
		}

		imageContent = openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL: dataURI,
			},
		}
	}

	// Create the vision request
	// Modify prompt to get a yes/no answer
	enhancedPrompt := req.Prompt + " Answer with 'yes' or 'no' first, then provide a brief explanation."

	resp, err := h.OpenAIClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "gpt-4o-mini",
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleUser,
					MultiContent: []openai.ChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeText,
							Text: enhancedPrompt,
						},
						imageContent,
					},
				},
			},
			MaxTokens: 300,
		},
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to analyze image: " + err.Error(),
		})
	}

	if len(resp.Choices) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "No response from OpenAI",
		})
	}

	answer := resp.Choices[0].Message.Content

	// Parse the answer to determine true/false
	result := false
	if len(answer) > 0 {
		// Check if the answer starts with "yes" (case-insensitive)
		firstWord := answer
		if len(answer) > 3 {
			firstWord = answer[:3]
		}
		if firstWord == "Yes" || firstWord == "yes" || firstWord == "YES" {
			result = true
		}
	}

	return c.JSON(ImageAnalysisResponse{
		Result: result,
	})
}
