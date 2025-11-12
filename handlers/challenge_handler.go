package handlers

import (
	"context"
	"fmt"

	"treasureHunt/models"
	"treasureHunt/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChallengeHandler struct {
	challengeRepo *repository.ChallengeRepository
	battleRepo    *repository.BattleRepository
	OpenAIClient  *openai.Client
}

func NewChallengeHandler(challengeRepo *repository.ChallengeRepository, openaiClient *openai.Client) *ChallengeHandler {
	return &ChallengeHandler{
		challengeRepo: challengeRepo,
		battleRepo:    nil, // Will be set later if needed
		OpenAIClient:  openaiClient,
	}
}

func (h *ChallengeHandler) SetBattleRepo(battleRepo *repository.BattleRepository) {
	h.battleRepo = battleRepo
}

type StartChallengeRequest struct {
	Difficulty string `json:"difficulty"`
	TotalItems int    `json:"totalItems"`
}

type VerifyItemRequest struct {
	ImageURL    string `json:"imageUrl,omitempty"`
	ImageBase64 string `json:"imageBase64,omitempty"`
}

// StartChallenge creates a new challenge
func (h *ChallengeHandler) StartChallenge(c *fiber.Ctx) error {
	// Get user ID from middleware
	userIDStr, ok := c.Locals("userID").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var req StartChallengeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validate difficulty
	difficulty := models.Difficulty(req.Difficulty)
	if difficulty != models.DifficultyEasy && difficulty != models.DifficultyMedium && difficulty != models.DifficultyHard {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid difficulty. Must be 'easy', 'medium', or 'hard'"})
	}

	// Validate total items
	if req.TotalItems <= 0 || req.TotalItems > 50 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Total items must be between 1 and 50"})
	}

	// Create challenge
	challenge, err := h.challengeRepo.CreateChallenge(context.Background(), userID, difficulty, req.TotalItems)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(challenge)
}

// GetChallenge retrieves a challenge by ID
func (h *ChallengeHandler) GetChallenge(c *fiber.Ctx) error {
	challengeIDStr := c.Params("challengeId")
	challengeID, err := primitive.ObjectIDFromHex(challengeIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid challenge ID"})
	}

	challenge, err := h.challengeRepo.GetChallenge(context.Background(), challengeID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	// Verify the challenge belongs to the user
	userIDStr, ok := c.Locals("userID").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	if challenge.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
	}

	return c.JSON(challenge)
}

// GetUserChallenges retrieves all challenges for the current user
func (h *ChallengeHandler) GetUserChallenges(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("userID").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	challenges, err := h.challengeRepo.GetUserChallenges(context.Background(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(challenges)
}

// VerifyItem verifies if an item is present in the uploaded image using OpenAI
func (h *ChallengeHandler) VerifyItem(c *fiber.Ctx) error {
	challengeIDStr := c.Params("challengeId")
	itemIDStr := c.Params("itemId")

	challengeID, err := primitive.ObjectIDFromHex(challengeIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid challenge ID"})
	}

	var itemID int
	if _, err := fmt.Sscanf(itemIDStr, "%d", &itemID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid item ID"})
	}

	// Verify challenge belongs to user
	userIDStr, ok := c.Locals("userID").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	challenge, err := h.challengeRepo.GetChallenge(context.Background(), challengeID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Challenge not found"})
	}

	if challenge.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
	}

	// Get the item
	item, err := h.challengeRepo.GetChallengeItem(context.Background(), challengeID, itemID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	// Check if item is already found
	if item.Found {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Item already found"})
	}

	// Parse request
	var req VerifyItemRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.ImageURL == "" && req.ImageBase64 == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Either imageUrl or imageBase64 is required"})
	}

	// Prepare the prompt
	prompt := fmt.Sprintf("Is there a %s in this image? Answer with just 'yes' or 'no' as the first word.", item.Name)

	// Create the message with image
	var imageURL string
	if req.ImageBase64 != "" {
		imageURL = req.ImageBase64
	} else {
		imageURL = req.ImageURL
	}

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
							Text: prompt,
						},
						{
							Type: openai.ChatMessagePartTypeImageURL,
							ImageURL: &openai.ChatMessageImageURL{
								URL: imageURL,
							},
						},
					},
				},
			},
			MaxTokens: 100,
		},
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to analyze image"})
	}

	if len(resp.Choices) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "No response from AI"})
	}

	answer := resp.Choices[0].Message.Content

	// Check if the answer contains "yes" (case-insensitive)
	found := false
	if len(answer) >= 3 {
		firstWord := answer[:3]
		if firstWord == "Yes" || firstWord == "yes" || firstWord == "YES" {
			found = true
		}
	}

	// If found, mark the item as found
	if found {
		updatedChallenge, err := h.challengeRepo.MarkItemFound(context.Background(), challengeID, itemID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		// Update battle progress if this challenge is part of a battle
		if h.battleRepo != nil {
			h.syncBattleProgress(userID, challengeID, updatedChallenge)
		}

		return c.JSON(fiber.Map{
			"found":     true,
			"message":   fmt.Sprintf("Great! You found the %s!", item.Name),
			"challenge": updatedChallenge,
		})
	}

	return c.JSON(fiber.Map{
		"found":   false,
		"message": fmt.Sprintf("No %s detected in the image. Try again!", item.Name),
	})
}

// syncBattleProgress updates battle progress when a challenge item is found
func (h *ChallengeHandler) syncBattleProgress(userID, challengeID primitive.ObjectID, challenge *models.Challenge) {
	if h.battleRepo == nil {
		return
	}

	// Find if this challenge is part of a battle
	battles, err := h.battleRepo.GetActiveBattles(userID)
	if err != nil {
		return
	}

	for _, battleWithUsers := range battles {
		// Check if any participant's challenge matches
		for _, participant := range battleWithUsers.Battle.Participants {
			if participant.UserID == userID && participant.ChallengeID == challengeID {
				// Update battle progress
				h.battleRepo.UpdateBattleProgress(
					battleWithUsers.Battle.ID,
					userID,
					challenge.CompletedItems,
					challenge.IsCompleted,
				)
				return
			}
		}
	}
}
