package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FriendRequestStatus string

const (
	FriendRequestPending  FriendRequestStatus = "pending"
	FriendRequestAccepted FriendRequestStatus = "accepted"
	FriendRequestRejected FriendRequestStatus = "rejected"
)

// FriendRequest represents a friend request sent from one user to another
type FriendRequest struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	FromUserID primitive.ObjectID  `bson:"fromUserId" json:"fromUserId"`
	ToUserID   primitive.ObjectID  `bson:"toUserId" json:"toUserId"`
	Status     FriendRequestStatus `bson:"status" json:"status"`
	CreatedAt  time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time           `bson:"updatedAt" json:"updatedAt"`
}

// Friendship represents an active friendship between two users
type Friendship struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	User1ID   primitive.ObjectID `bson:"user1Id" json:"user1Id"`
	User2ID   primitive.ObjectID `bson:"user2Id" json:"user2Id"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

// FriendRequestWithUser includes the user details
type FriendRequestWithUser struct {
	FriendRequest `bson:",inline"`
	FromUser      User `json:"fromUser" bson:"-"`
	ToUser        User `json:"toUser" bson:"-"`
}

// FriendWithStatus includes friend details and online status
type FriendWithStatus struct {
	User     User   `json:"user"`
	IsOnline bool   `json:"isOnline"`
	FriendID string `json:"friendId"`
}

// SendFriendRequestInput is the request payload for sending a friend request
type SendFriendRequestInput struct {
	Email string `json:"email" validate:"required,email"`
}
