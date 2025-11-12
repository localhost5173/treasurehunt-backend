package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationType string

const (
	NotificationFriendRequest NotificationType = "friend_request"
	NotificationBattleInvite  NotificationType = "battle_invite"
	NotificationBattleAccept  NotificationType = "battle_accept"
	NotificationFriendAccept  NotificationType = "friend_accept"
)

// Notification represents a notification for a user
type Notification struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"userId" json:"userId"`
	Type        NotificationType   `bson:"type" json:"type"`
	FromUser    primitive.ObjectID `bson:"fromUser" json:"fromUser"`
	ReferenceID primitive.ObjectID `bson:"referenceId,omitempty" json:"referenceId,omitempty"` // FriendRequest ID or Battle ID
	Message     string             `bson:"message" json:"message"`
	IsRead      bool               `bson:"isRead" json:"isRead"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
}

// NotificationWithUser includes the sender's user details
type NotificationWithUser struct {
	Notification `bson:",inline"`
	FromUserData User `json:"fromUserData" bson:"-"`
}
