package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string             `bson:"email" json:"email"`
	Password     string             `bson:"password,omitempty" json:"-"` // Hashed password, not returned in JSON
	Name         string             `bson:"name" json:"name"`
	GoogleID     string             `bson:"google_id,omitempty" json:"googleId,omitempty"`
	ProfileImage string             `bson:"profile_image,omitempty" json:"profileImage,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updatedAt"`
}

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
