package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

type ChallengeItem struct {
	ItemID  int       `json:"itemId" bson:"itemId"`
	Name    string    `json:"name" bson:"name"`
	Found   bool      `json:"found" bson:"found"`
	FoundAt time.Time `json:"foundAt,omitempty" bson:"foundAt,omitempty"`
}

type Challenge struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID         primitive.ObjectID `json:"userId" bson:"userId"`
	Difficulty     Difficulty         `json:"difficulty" bson:"difficulty"`
	TotalItems     int                `json:"totalItems" bson:"totalItems"`
	Items          []ChallengeItem    `json:"items" bson:"items"`
	CompletedItems int                `json:"completedItems" bson:"completedItems"`
	IsCompleted    bool               `json:"isCompleted" bson:"isCompleted"`
	CreatedAt      time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt" bson:"updatedAt"`
	CompletedAt    time.Time          `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
}
