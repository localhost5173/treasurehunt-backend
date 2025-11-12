package repository

import (
	"context"
	"errors"
	"time"
	"treasureHunt/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type FriendRepository struct {
	friendRequestCollection *mongo.Collection
	friendshipCollection    *mongo.Collection
	userCollection          *mongo.Collection
}

func NewFriendRepository(db *mongo.Database) *FriendRepository {
	return &FriendRepository{
		friendRequestCollection: db.Collection("friend_requests"),
		friendshipCollection:    db.Collection("friendships"),
		userCollection:          db.Collection("users"),
	}
}

// FindUserByEmail finds a user by email
func (r *FriendRepository) FindUserByEmail(email string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user models.User
	err := r.userCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// SendFriendRequest creates a new friend request
func (r *FriendRepository) SendFriendRequest(fromUserID, toUserID primitive.ObjectID) (*models.FriendRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if they are already friends
	friendship, _ := r.GetFriendship(fromUserID, toUserID)
	if friendship != nil {
		return nil, errors.New("already friends")
	}

	// Check if there's already a pending request
	existingRequest, _ := r.GetPendingRequest(fromUserID, toUserID)
	if existingRequest != nil {
		return nil, errors.New("friend request already exists")
	}

	// Check if there's a request from the other user
	reverseRequest, _ := r.GetPendingRequest(toUserID, fromUserID)
	if reverseRequest != nil {
		return nil, errors.New("this user has already sent you a friend request")
	}

	friendRequest := models.FriendRequest{
		ID:         primitive.NewObjectID(),
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Status:     models.FriendRequestPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	_, err := r.friendRequestCollection.InsertOne(ctx, friendRequest)
	if err != nil {
		return nil, err
	}

	return &friendRequest, nil
}

// GetPendingRequest finds a pending friend request between two users
func (r *FriendRepository) GetPendingRequest(fromUserID, toUserID primitive.ObjectID) (*models.FriendRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var request models.FriendRequest
	err := r.friendRequestCollection.FindOne(ctx, bson.M{
		"fromUserId": fromUserID,
		"toUserId":   toUserID,
		"status":     models.FriendRequestPending,
	}).Decode(&request)

	if err != nil {
		return nil, err
	}
	return &request, nil
}

// GetFriendRequests gets all pending friend requests for a user
func (r *FriendRepository) GetFriendRequests(userID primitive.ObjectID) ([]models.FriendRequestWithUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.friendRequestCollection.Find(ctx, bson.M{
		"toUserId": userID,
		"status":   models.FriendRequestPending,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var requests []models.FriendRequestWithUser
	for cursor.Next(ctx) {
		var request models.FriendRequest
		if err := cursor.Decode(&request); err != nil {
			continue
		}

		// Fetch the from user details
		var fromUser models.User
		err := r.userCollection.FindOne(ctx, bson.M{"_id": request.FromUserID}).Decode(&fromUser)
		if err != nil {
			continue
		}

		requests = append(requests, models.FriendRequestWithUser{
			FriendRequest: request,
			FromUser:      fromUser,
		})
	}

	return requests, nil
}

// AcceptFriendRequest accepts a friend request and creates a friendship
func (r *FriendRepository) AcceptFriendRequest(requestID, userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Find the friend request
	var request models.FriendRequest
	err := r.friendRequestCollection.FindOne(ctx, bson.M{
		"_id":      requestID,
		"toUserId": userID,
		"status":   models.FriendRequestPending,
	}).Decode(&request)
	if err != nil {
		return errors.New("friend request not found")
	}

	// Update the request status
	_, err = r.friendRequestCollection.UpdateOne(ctx, bson.M{"_id": requestID}, bson.M{
		"$set": bson.M{
			"status":    models.FriendRequestAccepted,
			"updatedAt": time.Now(),
		},
	})
	if err != nil {
		return err
	}

	// Create friendship
	friendship := models.Friendship{
		ID:        primitive.NewObjectID(),
		User1ID:   request.FromUserID,
		User2ID:   request.ToUserID,
		CreatedAt: time.Now(),
	}

	_, err = r.friendshipCollection.InsertOne(ctx, friendship)
	return err
}

// RejectFriendRequest rejects a friend request
func (r *FriendRepository) RejectFriendRequest(requestID, userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.friendRequestCollection.UpdateOne(ctx, bson.M{
		"_id":      requestID,
		"toUserId": userID,
		"status":   models.FriendRequestPending,
	}, bson.M{
		"$set": bson.M{
			"status":    models.FriendRequestRejected,
			"updatedAt": time.Now(),
		},
	})

	if err != nil {
		return err
	}

	if result.ModifiedCount == 0 {
		return errors.New("friend request not found")
	}

	return nil
}

// GetFriendship checks if two users are friends
func (r *FriendRepository) GetFriendship(user1ID, user2ID primitive.ObjectID) (*models.Friendship, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var friendship models.Friendship
	err := r.friendshipCollection.FindOne(ctx, bson.M{
		"$or": []bson.M{
			{"user1Id": user1ID, "user2Id": user2ID},
			{"user1Id": user2ID, "user2Id": user1ID},
		},
	}).Decode(&friendship)

	if err != nil {
		return nil, err
	}
	return &friendship, nil
}

// GetFriends gets all friends for a user with their online status
func (r *FriendRepository) GetFriends(userID primitive.ObjectID) ([]models.FriendWithStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.friendshipCollection.Find(ctx, bson.M{
		"$or": []bson.M{
			{"user1Id": userID},
			{"user2Id": userID},
		},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var friends []models.FriendWithStatus
	for cursor.Next(ctx) {
		var friendship models.Friendship
		if err := cursor.Decode(&friendship); err != nil {
			continue
		}

		// Get the friend's ID (the one that's not the current user)
		friendID := friendship.User1ID
		if friendID == userID {
			friendID = friendship.User2ID
		}

		// Fetch friend details
		var friend models.User
		err := r.userCollection.FindOne(ctx, bson.M{"_id": friendID}).Decode(&friend)
		if err != nil {
			continue
		}

		// Check if friend is online (last active within 5 minutes)
		isOnline := friend.IsOnline && time.Since(friend.LastActive) < 5*time.Minute

		friends = append(friends, models.FriendWithStatus{
			User:     friend,
			IsOnline: isOnline,
			FriendID: friendship.ID.Hex(),
		})
	}

	return friends, nil
}

// UpdateUserOnlineStatus updates the user's online status
func (r *FriendRepository) UpdateUserOnlineStatus(userID primitive.ObjectID, isOnline bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.userCollection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
		"$set": bson.M{
			"isOnline":   isOnline,
			"lastActive": time.Now(),
		},
	})

	return err
}
