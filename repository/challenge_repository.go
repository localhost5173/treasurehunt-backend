package repository

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"treasureHunt/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	// Item pools for different difficulty levels
	EasyItems = []string{
		"car", "pinecone", "stone", "bicycle", "street light", "house",
		"stick", "acorn", "trashcan", "bush", "fence", "crossroad", "football goal",
	}

	MediumItems = []string{
		"moped", "bench", "traffic light", "truck", "bus", "playground",
		"bus stop", "dog",
	}

	HardItems = []string{
		"creek", "toadstool", "bird", "worm", "boat", "snail", "bee", "taxi", "Kanye West",
	}
)

type ChallengeRepository struct {
	collection *mongo.Collection
}

func NewChallengeRepository(db *mongo.Database) *ChallengeRepository {
	return &ChallengeRepository{
		collection: db.Collection("challenges"),
	}
}

// CreateChallenge creates a new challenge with random items based on difficulty
func (r *ChallengeRepository) CreateChallenge(ctx context.Context, userID primitive.ObjectID, difficulty models.Difficulty, totalItems int) (*models.Challenge, error) {
	return r.CreateChallengeWithBattle(ctx, userID, difficulty, totalItems, nil)
}

// CreateChallengeWithBattle creates a new challenge with optional battleId
func (r *ChallengeRepository) CreateChallengeWithBattle(ctx context.Context, userID primitive.ObjectID, difficulty models.Difficulty, totalItems int, battleID *primitive.ObjectID) (*models.Challenge, error) {
	if totalItems <= 0 || totalItems > 50 {
		return nil, errors.New("total items must be between 1 and 50")
	}

	// Define difficulty configurations: map[totalItems] -> [easyCount, mediumCount, hardCount]
	difficultyConfigs := map[models.Difficulty]map[int][]int{
		models.DifficultyEasy: {
			3:  {3, 0, 0},
			5:  {5, 0, 0},
			10: {10, 0, 0},
		},
		models.DifficultyMedium: {
			3:  {2, 1, 0},
			5:  {3, 2, 0},
			10: {6, 4, 0},
		},
		models.DifficultyHard: {
			3:  {1, 1, 1},
			5:  {1, 2, 2},
			10: {2, 4, 4},
		},
	}

	// Get the configuration for this difficulty and item count
	config, exists := difficultyConfigs[difficulty][totalItems]
	if !exists {
		// Default fallback: distribute items based on difficulty
		switch difficulty {
		case models.DifficultyEasy:
			config = []int{totalItems, 0, 0}
		case models.DifficultyMedium:
			config = []int{totalItems / 2, totalItems - totalItems/2, 0}
		case models.DifficultyHard:
			config = []int{totalItems / 3, totalItems / 3, totalItems - 2*(totalItems/3)}
		default:
			return nil, errors.New("invalid difficulty level")
		}
	}

	easyCount := config[0]
	mediumCount := config[1]
	hardCount := config[2]

	// Randomly select items from each pool without duplicates
	items := make([]models.ChallengeItem, 0, totalItems)
	rand.Seed(time.Now().UnixNano())

	// Helper function to select unique items from a pool
	selectUniqueItems := func(pool []string, count int) []string {
		if count > len(pool) {
			count = len(pool)
		}
		// Create a copy of the pool and shuffle it
		poolCopy := make([]string, len(pool))
		copy(poolCopy, pool)
		rand.Shuffle(len(poolCopy), func(i, j int) {
			poolCopy[i], poolCopy[j] = poolCopy[j], poolCopy[i]
		})
		// Take the first 'count' items
		return poolCopy[:count]
	}

	// Select easy items (no duplicates)
	selectedEasy := selectUniqueItems(EasyItems, easyCount)
	for _, itemName := range selectedEasy {
		items = append(items, models.ChallengeItem{
			ItemID: len(items) + 1,
			Name:   itemName,
			Found:  false,
		})
	}

	// Select medium items (no duplicates)
	selectedMedium := selectUniqueItems(MediumItems, mediumCount)
	for _, itemName := range selectedMedium {
		items = append(items, models.ChallengeItem{
			ItemID: len(items) + 1,
			Name:   itemName,
			Found:  false,
		})
	}

	// Select hard items (no duplicates)
	selectedHard := selectUniqueItems(HardItems, hardCount)
	for _, itemName := range selectedHard {
		items = append(items, models.ChallengeItem{
			ItemID: len(items) + 1,
			Name:   itemName,
			Found:  false,
		})
	}

	// Shuffle the items to mix difficulties
	rand.Shuffle(len(items), func(i, j int) {
		items[i], items[j] = items[j], items[i]
	})

	// Re-number items after shuffle
	for i := range items {
		items[i].ItemID = i + 1
	}

	challenge := &models.Challenge{
		UserID:         userID,
		BattleID:       battleID,
		Difficulty:     difficulty,
		TotalItems:     len(items),
		Items:          items,
		CompletedItems: 0,
		IsCompleted:    false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	result, err := r.collection.InsertOne(ctx, challenge)
	if err != nil {
		return nil, err
	}

	challenge.ID = result.InsertedID.(primitive.ObjectID)
	return challenge, nil
}

// CreateChallengeWithItems creates a challenge using specific items (for battles)
func (r *ChallengeRepository) CreateChallengeWithItems(ctx context.Context, userID primitive.ObjectID, difficulty models.Difficulty, items []models.ChallengeItem, battleID *primitive.ObjectID) (*models.Challenge, error) {
	// Create a copy of items with Found set to false for this user
	userItems := make([]models.ChallengeItem, len(items))
	for i, item := range items {
		userItems[i] = models.ChallengeItem{
			ItemID: item.ItemID,
			Name:   item.Name,
			Found:  false,
		}
	}

	challenge := &models.Challenge{
		UserID:         userID,
		BattleID:       battleID,
		Difficulty:     difficulty,
		TotalItems:     len(userItems),
		Items:          userItems,
		CompletedItems: 0,
		IsCompleted:    false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	result, err := r.collection.InsertOne(ctx, challenge)
	if err != nil {
		return nil, err
	}

	challenge.ID = result.InsertedID.(primitive.ObjectID)
	return challenge, nil
}

// GetChallenge retrieves a challenge by ID
func (r *ChallengeRepository) GetChallenge(ctx context.Context, challengeID primitive.ObjectID) (*models.Challenge, error) {
	var challenge models.Challenge
	err := r.collection.FindOne(ctx, bson.M{"_id": challengeID}).Decode(&challenge)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("challenge not found")
		}
		return nil, err
	}
	return &challenge, nil
}

// GetUserChallenges retrieves all challenges for a user
func (r *ChallengeRepository) GetUserChallenges(ctx context.Context, userID primitive.ObjectID) ([]models.Challenge, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var challenges []models.Challenge
	if err := cursor.All(ctx, &challenges); err != nil {
		return nil, err
	}

	return challenges, nil
}

// MarkItemFound marks an item as found in a challenge
func (r *ChallengeRepository) MarkItemFound(ctx context.Context, challengeID primitive.ObjectID, itemID int) (*models.Challenge, error) {
	challenge, err := r.GetChallenge(ctx, challengeID)
	if err != nil {
		return nil, err
	}

	// Find the item and mark it as found
	itemFound := false
	for i := range challenge.Items {
		if challenge.Items[i].ItemID == itemID {
			if challenge.Items[i].Found {
				return nil, errors.New("item already found")
			}
			challenge.Items[i].Found = true
			challenge.Items[i].FoundAt = time.Now()
			itemFound = true
			break
		}
	}

	if !itemFound {
		return nil, errors.New("item not found in challenge")
	}

	// Update completed items count
	challenge.CompletedItems++
	challenge.UpdatedAt = time.Now()

	// Check if challenge is completed
	if challenge.CompletedItems >= challenge.TotalItems {
		challenge.IsCompleted = true
		challenge.CompletedAt = time.Now()
	}

	// Update in database
	update := bson.M{
		"$set": bson.M{
			"items":          challenge.Items,
			"completedItems": challenge.CompletedItems,
			"isCompleted":    challenge.IsCompleted,
			"updatedAt":      challenge.UpdatedAt,
			"completedAt":    challenge.CompletedAt,
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": challengeID}, update)
	if err != nil {
		return nil, err
	}

	return challenge, nil
}

// GetChallengeItem retrieves a specific item from a challenge
func (r *ChallengeRepository) GetChallengeItem(ctx context.Context, challengeID primitive.ObjectID, itemID int) (*models.ChallengeItem, error) {
	challenge, err := r.GetChallenge(ctx, challengeID)
	if err != nil {
		return nil, err
	}

	for _, item := range challenge.Items {
		if item.ItemID == itemID {
			return &item, nil
		}
	}

	return nil, errors.New("item not found in challenge")
}
