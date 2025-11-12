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

type BattleRepository struct {
	battleCollection    *mongo.Collection
	challengeCollection *mongo.Collection
	userCollection      *mongo.Collection
}

func NewBattleRepository(db *mongo.Database) *BattleRepository {
	return &BattleRepository{
		battleCollection:    db.Collection("battles"),
		challengeCollection: db.Collection("challenges"),
		userCollection:      db.Collection("users"),
	}
}

// CreateBattle creates a new battle between two users
func (r *BattleRepository) CreateBattle(challengerID, opponentID primitive.ObjectID, difficulty models.Difficulty, totalItems int, items []models.ChallengeItem) (*models.Battle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	battle := models.Battle{
		ID:           primitive.NewObjectID(),
		ChallengerID: challengerID,
		OpponentID:   opponentID,
		Difficulty:   difficulty,
		TotalItems:   totalItems,
		Items:        items,
		Participants: []models.BattleParticipant{
			{
				UserID:         challengerID,
				CompletedItems: 0,
				IsCompleted:    false,
				HasAccepted:    true, // Challenger auto-accepts
			},
			{
				UserID:         opponentID,
				CompletedItems: 0,
				IsCompleted:    false,
				HasAccepted:    false, // Opponent needs to accept
			},
		},
		Status:    models.BattleStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := r.battleCollection.InsertOne(ctx, battle)
	if err != nil {
		return nil, err
	}

	return &battle, nil
}

// GetBattle retrieves a battle by ID
func (r *BattleRepository) GetBattle(battleID primitive.ObjectID) (*models.Battle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var battle models.Battle
	err := r.battleCollection.FindOne(ctx, bson.M{"_id": battleID}).Decode(&battle)
	if err != nil {
		return nil, err
	}

	return &battle, nil
}

// GetBattleWithUsers retrieves a battle with user details
func (r *BattleRepository) GetBattleWithUsers(battleID primitive.ObjectID) (*models.BattleWithUsers, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	battle, err := r.GetBattle(battleID)
	if err != nil {
		return nil, err
	}

	var challenger, opponent models.User
	err = r.userCollection.FindOne(ctx, bson.M{"_id": battle.ChallengerID}).Decode(&challenger)
	if err != nil {
		return nil, err
	}

	err = r.userCollection.FindOne(ctx, bson.M{"_id": battle.OpponentID}).Decode(&opponent)
	if err != nil {
		return nil, err
	}

	return &models.BattleWithUsers{
		Battle:     *battle,
		Challenger: challenger,
		Opponent:   opponent,
	}, nil
}

// GetUserBattles gets all battles for a user
func (r *BattleRepository) GetUserBattles(userID primitive.ObjectID) ([]models.BattleWithUsers, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.battleCollection.Find(ctx, bson.M{
		"$or": []bson.M{
			{"challengerId": userID},
			{"opponentId": userID},
		},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var battles []models.BattleWithUsers
	for cursor.Next(ctx) {
		var battle models.Battle
		if err := cursor.Decode(&battle); err != nil {
			continue
		}

		var challenger, opponent models.User
		r.userCollection.FindOne(ctx, bson.M{"_id": battle.ChallengerID}).Decode(&challenger)
		r.userCollection.FindOne(ctx, bson.M{"_id": battle.OpponentID}).Decode(&opponent)

		battles = append(battles, models.BattleWithUsers{
			Battle:     battle,
			Challenger: challenger,
			Opponent:   opponent,
		})
	}

	return battles, nil
}

// AcceptBattle accepts a battle invitation
func (r *BattleRepository) AcceptBattle(battleID, userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	battle, err := r.GetBattle(battleID)
	if err != nil {
		return err
	}

	if battle.OpponentID != userID {
		return errors.New("you are not the opponent of this battle")
	}

	if battle.Status != models.BattleStatusPending {
		return errors.New("battle is not pending")
	}

	// Update participant acceptance and battle status
	_, err = r.battleCollection.UpdateOne(ctx, bson.M{
		"_id":                 battleID,
		"participants.userId": userID,
	}, bson.M{
		"$set": bson.M{
			"participants.$.hasAccepted": true,
			"status":                     models.BattleStatusActive,
			"updatedAt":                  time.Now(),
		},
	})

	return err
}

// DeclineBattle declines a battle invitation
func (r *BattleRepository) DeclineBattle(battleID, userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	battle, err := r.GetBattle(battleID)
	if err != nil {
		return err
	}

	if battle.OpponentID != userID {
		return errors.New("you are not the opponent of this battle")
	}

	if battle.Status != models.BattleStatusPending {
		return errors.New("battle is not pending")
	}

	_, err = r.battleCollection.UpdateOne(ctx, bson.M{"_id": battleID}, bson.M{
		"$set": bson.M{
			"status":    models.BattleStatusDeclined,
			"updatedAt": time.Now(),
		},
	})

	return err
}

// LinkChallengeToParticipant links a challenge to a battle participant
func (r *BattleRepository) LinkChallengeToParticipant(battleID, userID, challengeID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.battleCollection.UpdateOne(ctx, bson.M{
		"_id":                 battleID,
		"participants.userId": userID,
	}, bson.M{
		"$set": bson.M{
			"participants.$.challengeId": challengeID,
			"updatedAt":                  time.Now(),
		},
	})

	return err
}

// UpdateBattleProgress updates a participant's progress in a battle
func (r *BattleRepository) UpdateBattleProgress(battleID, userID primitive.ObjectID, completedItems int, isCompleted bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	updateFields := bson.M{
		"participants.$.completedItems": completedItems,
		"participants.$.isCompleted":    isCompleted,
		"updatedAt":                     time.Now(),
	}

	if isCompleted {
		updateFields["participants.$.completedAt"] = time.Now()
	}

	_, err := r.battleCollection.UpdateOne(ctx, bson.M{
		"_id":                 battleID,
		"participants.userId": userID,
	}, bson.M{
		"$set": updateFields,
	})

	if err != nil {
		return err
	}

	// Check if battle is complete and determine winner
	battle, err := r.GetBattle(battleID)
	if err != nil {
		return err
	}

	// Check if both participants completed
	allCompleted := true
	var winner primitive.ObjectID
	var winnerCompletedAt time.Time

	for _, p := range battle.Participants {
		if !p.IsCompleted {
			allCompleted = false
			break
		}
		// Find the winner (first to complete)
		if winner.IsZero() || (!p.CompletedAt.IsZero() && p.CompletedAt.Before(winnerCompletedAt)) {
			winner = p.UserID
			winnerCompletedAt = p.CompletedAt
		}
	}

	if allCompleted {
		_, err = r.battleCollection.UpdateOne(ctx, bson.M{"_id": battleID}, bson.M{
			"$set": bson.M{
				"status":      models.BattleStatusCompleted,
				"winnerId":    winner,
				"completedAt": time.Now(),
				"updatedAt":   time.Now(),
			},
		})
	}

	return err
}

// GetActiveBattles gets all active battles for a user
func (r *BattleRepository) GetActiveBattles(userID primitive.ObjectID) ([]models.BattleWithUsers, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.battleCollection.Find(ctx, bson.M{
		"$or": []bson.M{
			{"challengerId": userID},
			{"opponentId": userID},
		},
		"status": bson.M{
			"$in": []models.BattleStatus{models.BattleStatusPending, models.BattleStatusActive},
		},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var battles []models.BattleWithUsers
	for cursor.Next(ctx) {
		var battle models.Battle
		if err := cursor.Decode(&battle); err != nil {
			continue
		}

		var challenger, opponent models.User
		r.userCollection.FindOne(ctx, bson.M{"_id": battle.ChallengerID}).Decode(&challenger)
		r.userCollection.FindOne(ctx, bson.M{"_id": battle.OpponentID}).Decode(&opponent)

		battles = append(battles, models.BattleWithUsers{
			Battle:     battle,
			Challenger: challenger,
			Opponent:   opponent,
		})
	}

	return battles, nil
}
