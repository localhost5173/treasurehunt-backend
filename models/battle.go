package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BattleStatus string

const (
	BattleStatusPending   BattleStatus = "pending"
	BattleStatusActive    BattleStatus = "active"
	BattleStatusCompleted BattleStatus = "completed"
	BattleStatusDeclined  BattleStatus = "declined"
)

// BattleParticipant represents a player in a battle
type BattleParticipant struct {
	UserID         primitive.ObjectID `bson:"userId" json:"userId"`
	ChallengeID    primitive.ObjectID `bson:"challengeId,omitempty" json:"challengeId,omitempty"`
	CompletedItems int                `bson:"completedItems" json:"completedItems"`
	IsCompleted    bool               `bson:"isCompleted" json:"isCompleted"`
	CompletedAt    time.Time          `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	HasAccepted    bool               `bson:"hasAccepted" json:"hasAccepted"`
}

// Battle represents a competitive challenge between two friends
type Battle struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ChallengerID primitive.ObjectID  `bson:"challengerId" json:"challengerId"`
	OpponentID   primitive.ObjectID  `bson:"opponentId" json:"opponentId"`
	Difficulty   Difficulty          `bson:"difficulty" json:"difficulty"`
	TotalItems   int                 `bson:"totalItems" json:"totalItems"`
	Items        []ChallengeItem     `bson:"items" json:"items"`
	Participants []BattleParticipant `bson:"participants" json:"participants"`
	Status       BattleStatus        `bson:"status" json:"status"`
	WinnerID     primitive.ObjectID  `bson:"winnerId,omitempty" json:"winnerId,omitempty"`
	CreatedAt    time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time           `bson:"updatedAt" json:"updatedAt"`
	CompletedAt  time.Time           `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
}

// BattleWithUsers includes the user details for both participants
type BattleWithUsers struct {
	Battle     `bson:",inline"`
	Challenger User `json:"challenger" bson:"-"`
	Opponent   User `json:"opponent" bson:"-"`
}

// CreateBattleInput is the request payload for creating a battle
type CreateBattleInput struct {
	OpponentID string     `json:"opponentId" validate:"required"`
	Difficulty Difficulty `json:"difficulty" validate:"required,oneof=easy medium hard"`
	TotalItems int        `json:"totalItems" validate:"required,min=5,max=20"`
}
