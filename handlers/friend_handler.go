package handlers

import (
	"fmt"
	"treasureHunt/models"
	"treasureHunt/repository"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FriendHandler struct {
	friendRepo       *repository.FriendRepository
	notificationRepo *repository.NotificationRepository
}

func NewFriendHandler(friendRepo *repository.FriendRepository, notificationRepo *repository.NotificationRepository) *FriendHandler {
	return &FriendHandler{
		friendRepo:       friendRepo,
		notificationRepo: notificationRepo,
	}
}

// SendFriendRequest sends a friend request to a user by email
func (h *FriendHandler) SendFriendRequest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	fromUserID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var input models.SendFriendRequestInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Find user by email
	toUser, err := h.friendRepo.FindUserByEmail(input.Email)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Can't send friend request to yourself
	if toUser.ID == fromUserID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot send friend request to yourself"})
	}

	// Send friend request
	friendRequest, err := h.friendRepo.SendFriendRequest(fromUserID, toUser.ID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Create notification
	message := "sent you a friend request"
	h.notificationRepo.CreateNotification(
		toUser.ID,
		fromUserID,
		friendRequest.ID,
		models.NotificationFriendRequest,
		message,
	)

	return c.Status(fiber.StatusCreated).JSON(friendRequest)
}

// GetFriendRequests gets all pending friend requests for the current user
func (h *FriendHandler) GetFriendRequests(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	requests, err := h.friendRepo.GetFriendRequests(objID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch friend requests"})
	}

	if requests == nil {
		requests = []models.FriendRequestWithUser{}
	}

	return c.JSON(requests)
}

// AcceptFriendRequest accepts a friend request
func (h *FriendHandler) AcceptFriendRequest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	requestID := c.Params("requestId")
	reqObjID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request ID"})
	}

	err = h.friendRepo.AcceptFriendRequest(reqObjID, objID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Get the friend request to find the from user
	// Create notification for the sender that their request was accepted
	// This would require fetching the request first to get fromUserID
	// For now, simplified

	return c.JSON(fiber.Map{"message": "Friend request accepted"})
}

// RejectFriendRequest rejects a friend request
func (h *FriendHandler) RejectFriendRequest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	requestID := c.Params("requestId")
	reqObjID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request ID"})
	}

	err = h.friendRepo.RejectFriendRequest(reqObjID, objID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Friend request rejected"})
}

// GetFriends gets all friends for the current user
func (h *FriendHandler) GetFriends(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	friends, err := h.friendRepo.GetFriends(objID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch friends"})
	}

	if friends == nil {
		friends = []models.FriendWithStatus{}
	}

	return c.JSON(friends)
}

// UpdateOnlineStatus updates the user's online status
func (h *FriendHandler) UpdateOnlineStatus(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var input struct {
		IsOnline bool `json:"isOnline"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	err = h.friendRepo.UpdateUserOnlineStatus(objID, input.IsOnline)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update online status"})
	}

	return c.JSON(fiber.Map{"message": "Online status updated"})
}

// SearchUserByEmail searches for a user by email (for adding friends)
func (h *FriendHandler) SearchUserByEmail(c *fiber.Ctx) error {
	email := c.Query("email")
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email is required"})
	}

	user, err := h.friendRepo.FindUserByEmail(email)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Return limited user info
	return c.JSON(fiber.Map{
		"id":    user.ID.Hex(),
		"email": user.Email,
		"name":  user.Name,
	})
}

// GetNotifications gets all notifications for the current user
func (h *FriendHandler) GetNotifications(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	unreadOnly := c.Query("unreadOnly") == "true"

	var notifications []models.NotificationWithUser
	if unreadOnly {
		notifications, err = h.notificationRepo.GetUnreadNotifications(objID)
	} else {
		notifications, err = h.notificationRepo.GetUserNotifications(objID)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch notifications"})
	}

	if notifications == nil {
		notifications = []models.NotificationWithUser{}
	}

	return c.JSON(notifications)
}

// MarkNotificationRead marks a notification as read
func (h *FriendHandler) MarkNotificationRead(c *fiber.Ctx) error {
	notificationID := c.Params("notificationId")
	objID, err := primitive.ObjectIDFromHex(notificationID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid notification ID"})
	}

	err = h.notificationRepo.MarkNotificationAsRead(objID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to mark notification as read"})
	}

	return c.JSON(fiber.Map{"message": "Notification marked as read"})
}

// MarkAllNotificationsRead marks all notifications as read
func (h *FriendHandler) MarkAllNotificationsRead(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	err = h.notificationRepo.MarkAllNotificationsAsRead(objID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to mark all notifications as read"})
	}

	return c.JSON(fiber.Map{"message": "All notifications marked as read"})
}

// GetUnreadCount gets the count of unread notifications
func (h *FriendHandler) GetUnreadCount(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	count, err := h.notificationRepo.GetUnreadCount(objID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get unread count"})
	}

	return c.JSON(fiber.Map{"count": fmt.Sprint(count)})
}
