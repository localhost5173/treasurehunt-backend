package handlers

import (
	"context"
	"log"
	"math/rand"
	"time"
	"treasureHunt/models"
	"treasureHunt/repository"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BattleHandler struct {
	battleRepo       *repository.BattleRepository
	friendRepo       *repository.FriendRepository
	notificationRepo *repository.NotificationRepository
	challengeRepo    *repository.ChallengeRepository
	wsHub            *Hub
}

func NewBattleHandler(
	battleRepo *repository.BattleRepository,
	friendRepo *repository.FriendRepository,
	notificationRepo *repository.NotificationRepository,
	challengeRepo *repository.ChallengeRepository,
	wsHub *Hub,
) *BattleHandler {
	return &BattleHandler{
		battleRepo:       battleRepo,
		friendRepo:       friendRepo,
		notificationRepo: notificationRepo,
		challengeRepo:    challengeRepo,
		wsHub:            wsHub,
	}
}

// itemPool represents the available items for treasure hunts
var itemPool = []string{
	"apple", "banana", "orange", "grape", "watermelon",
	"book", "pen", "pencil", "notebook", "eraser",
	"shoe", "hat", "sunglasses", "watch", "backpack",
	"phone", "laptop", "keyboard", "mouse", "headphones",
	"cup", "plate", "fork", "spoon", "bottle",
	"chair", "table", "lamp", "pillow", "blanket",
	"car", "bicycle", "skateboard", "ball", "toy",
	"flower", "tree", "rock", "leaf", "shell",
	"dog", "cat", "bird", "fish", "hamster",
}

func generateChallengeItems(totalItems int) []models.ChallengeItem {
	rand.Seed(time.Now().UnixNano())

	// Shuffle the item pool
	shuffled := make([]string, len(itemPool))
	copy(shuffled, itemPool)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	items := make([]models.ChallengeItem, totalItems)
	for i := 0; i < totalItems; i++ {
		items[i] = models.ChallengeItem{
			ItemID: i + 1,
			Name:   shuffled[i%len(shuffled)],
			Found:  false,
		}
	}

	return items
}

// CreateBattle creates a new battle challenge
func (h *BattleHandler) CreateBattle(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	challengerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var input models.CreateBattleInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	opponentID, err := primitive.ObjectIDFromHex(input.OpponentID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid opponent ID"})
	}

	// Check if users are friends
	friendship, err := h.friendRepo.GetFriendship(challengerID, opponentID)
	if err != nil || friendship == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "You can only battle with friends"})
	}

	// Generate challenge items
	items := generateChallengeItems(input.TotalItems)

	// Create the battle
	battle, err := h.battleRepo.CreateBattle(challengerID, opponentID, input.Difficulty, input.TotalItems, items)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create battle"})
	}

	// Create notification for opponent
	message := "challenged you to a battle!"
	notification, err := h.notificationRepo.CreateNotification(
		opponentID,
		challengerID,
		battle.ID,
		models.NotificationBattleInvite,
		message,
	)
	if err != nil {
		log.Printf("Failed to create notification: %v", err)
	}

	// Send real-time notification via WebSocket
	if notification != nil && h.wsHub != nil {
		if err := h.wsHub.SendToUser(opponentID, notification); err != nil {
			log.Printf("Failed to send WebSocket notification: %v", err)
		}

		// Send battle list refresh message to opponent
		h.wsHub.SendToUserWithType(opponentID, "battle_list_update", map[string]interface{}{
			"action": "refresh_battles",
		})
	}

	// Return battle with user details
	battleWithUsers, err := h.battleRepo.GetBattleWithUsers(battle.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch battle details"})
	}

	return c.Status(fiber.StatusCreated).JSON(battleWithUsers)
}

// GetBattle retrieves a specific battle
func (h *BattleHandler) GetBattle(c *fiber.Ctx) error {
	battleID := c.Params("battleId")
	objID, err := primitive.ObjectIDFromHex(battleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid battle ID"})
	}

	battle, err := h.battleRepo.GetBattleWithUsers(objID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Battle not found"})
	}

	return c.JSON(battle)
}

// GetUserBattles gets all battles for the current user
func (h *BattleHandler) GetUserBattles(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	battles, err := h.battleRepo.GetUserBattles(objID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch battles"})
	}

	if battles == nil {
		battles = []models.BattleWithUsers{}
	}

	return c.JSON(battles)
}

// GetActiveBattles gets all active battles for the current user
func (h *BattleHandler) GetActiveBattles(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	battles, err := h.battleRepo.GetActiveBattles(objID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch active battles"})
	}

	if battles == nil {
		battles = []models.BattleWithUsers{}
	}

	return c.JSON(battles)
}

// AcceptBattle accepts a battle invitation
func (h *BattleHandler) AcceptBattle(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	battleID := c.Params("battleId")
	battleObjID, err := primitive.ObjectIDFromHex(battleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid battle ID"})
	}

	err = h.battleRepo.AcceptBattle(battleObjID, objID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Get battle details to notify the challenger
	battle, _ := h.battleRepo.GetBattle(battleObjID)
	if battle != nil {
		message := "accepted your battle challenge!"
		notification, err := h.notificationRepo.CreateNotification(
			battle.ChallengerID,
			objID,
			battleObjID,
			models.NotificationBattleAccept,
			message,
		)
		if err != nil {
			log.Printf("Failed to create notification: %v", err)
		}

		// Send real-time notification via WebSocket
		if notification != nil && h.wsHub != nil {
			if err := h.wsHub.SendToUser(battle.ChallengerID, notification); err != nil {
				log.Printf("Failed to send WebSocket notification: %v", err)
			}

			// Send battle_started event to the challenger so they get redirected to the challenge
			h.wsHub.SendToUserWithType(battle.ChallengerID, "battle_started", map[string]interface{}{
				"battleId": battleObjID.Hex(),
				"action":   "start_battle",
			})

			// Send battle list refresh message to both users
			h.wsHub.SendToUserWithType(battle.ChallengerID, "battle_list_update", map[string]interface{}{
				"action": "refresh_battles",
			})
			h.wsHub.SendToUserWithType(objID, "battle_list_update", map[string]interface{}{
				"action": "refresh_battles",
			})
		}
	}

	return c.JSON(fiber.Map{"message": "Battle accepted"})
}

// DeclineBattle declines a battle invitation
func (h *BattleHandler) DeclineBattle(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	battleID := c.Params("battleId")
	battleObjID, err := primitive.ObjectIDFromHex(battleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid battle ID"})
	}

	err = h.battleRepo.DeclineBattle(battleObjID, objID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Get battle details to notify the challenger about decline
	battle, _ := h.battleRepo.GetBattle(battleObjID)
	if battle != nil && h.wsHub != nil {
		// Send battle list refresh to challenger so declined battle updates
		h.wsHub.SendToUserWithType(battle.ChallengerID, "battle_list_update", map[string]interface{}{
			"action": "refresh_battles",
		})
	}

	return c.JSON(fiber.Map{"message": "Battle declined"})
}

// JoinBattle creates or retrieves a challenge for the user in a battle
func (h *BattleHandler) JoinBattle(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	battleID := c.Params("battleId")
	battleObjID, err := primitive.ObjectIDFromHex(battleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid battle ID"})
	}

	battle, err := h.battleRepo.GetBattle(battleObjID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Battle not found"})
	}

	// Check if user is part of this battle
	isParticipant := false
	var participant *models.BattleParticipant
	for i, p := range battle.Participants {
		if p.UserID == userObjID {
			isParticipant = true
			participant = &battle.Participants[i]
			break
		}
	}

	if !isParticipant {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You are not part of this battle"})
	}

	// If user already has a challenge linked, return it
	if !participant.ChallengeID.IsZero() {
		ctx := context.Background()
		challenge, err := h.challengeRepo.GetChallenge(ctx, participant.ChallengeID)
		if err == nil {
			return c.JSON(challenge)
		}
	}

	// Create a new challenge for this user based on the battle's items
	// Use the battle's items to ensure both players have the same challenge items
	ctx := context.Background()
	challenge, err := h.challengeRepo.CreateChallengeWithItems(ctx, userObjID, battle.Difficulty, battle.Items, &battleObjID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create challenge"})
	}

	// Link the challenge to the battle participant
	err = h.battleRepo.LinkChallengeToParticipant(battleObjID, userObjID, challenge.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to link challenge to battle"})
	}

	return c.JSON(challenge)
}
