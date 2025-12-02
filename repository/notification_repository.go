package repository

import (
	"context"
	"time"
	"treasureHunt/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type NotificationRepository struct {
	notificationCollection *mongo.Collection
	userCollection         *mongo.Collection
}

func NewNotificationRepository(db *mongo.Database) *NotificationRepository {
	return &NotificationRepository{
		notificationCollection: db.Collection("notifications"),
		userCollection:         db.Collection("users"),
	}
}

// CreateNotification creates a new notification
func (r *NotificationRepository) CreateNotification(userID, fromUser, referenceID primitive.ObjectID, notifType models.NotificationType, message string) (*models.NotificationWithUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	notification := models.Notification{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		Type:        notifType,
		FromUser:    fromUser,
		ReferenceID: referenceID,
		Message:     message,
		IsRead:      false,
		CreatedAt:   time.Now(),
	}

	_, err := r.notificationCollection.InsertOne(ctx, notification)
	if err != nil {
		return nil, err
	}

	// Fetch the from user details
	var fromUserData models.User
	err = r.userCollection.FindOne(ctx, bson.M{"_id": fromUser}).Decode(&fromUserData)
	if err != nil {
		// Return notification even if we can't fetch user data
		return &models.NotificationWithUser{
			Notification: notification,
		}, nil
	}

	return &models.NotificationWithUser{
		Notification: notification,
		FromUserData: fromUserData,
	}, nil
}

// GetUserNotifications gets all notifications for a user
func (r *NotificationRepository) GetUserNotifications(userID primitive.ObjectID) ([]models.NotificationWithUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := r.notificationCollection.Find(ctx, bson.M{"userId": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []models.NotificationWithUser
	for cursor.Next(ctx) {
		var notification models.Notification
		if err := cursor.Decode(&notification); err != nil {
			continue
		}

		// Fetch the from user details
		var fromUser models.User
		err := r.userCollection.FindOne(ctx, bson.M{"_id": notification.FromUser}).Decode(&fromUser)
		if err != nil {
			continue
		}

		notifications = append(notifications, models.NotificationWithUser{
			Notification: notification,
			FromUserData: fromUser,
		})
	}

	return notifications, nil
}

// GetUnreadNotifications gets all unread notifications for a user
func (r *NotificationRepository) GetUnreadNotifications(userID primitive.ObjectID) ([]models.NotificationWithUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := r.notificationCollection.Find(ctx, bson.M{
		"userId": userID,
		"isRead": false,
	}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []models.NotificationWithUser
	for cursor.Next(ctx) {
		var notification models.Notification
		if err := cursor.Decode(&notification); err != nil {
			continue
		}

		// Fetch the from user details
		var fromUser models.User
		err := r.userCollection.FindOne(ctx, bson.M{"_id": notification.FromUser}).Decode(&fromUser)
		if err != nil {
			continue
		}

		notifications = append(notifications, models.NotificationWithUser{
			Notification: notification,
			FromUserData: fromUser,
		})
	}

	return notifications, nil
}

// MarkNotificationAsRead marks a notification as read
func (r *NotificationRepository) MarkNotificationAsRead(notificationID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.notificationCollection.UpdateOne(ctx, bson.M{"_id": notificationID}, bson.M{
		"$set": bson.M{"isRead": true},
	})

	return err
}

// MarkAllNotificationsAsRead marks all notifications for a user as read
func (r *NotificationRepository) MarkAllNotificationsAsRead(userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.notificationCollection.UpdateMany(ctx, bson.M{
		"userId": userID,
		"isRead": false,
	}, bson.M{
		"$set": bson.M{"isRead": true},
	})

	return err
}

// DeleteNotification deletes a notification
func (r *NotificationRepository) DeleteNotification(notificationID, userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.notificationCollection.DeleteOne(ctx, bson.M{
		"_id":    notificationID,
		"userId": userID,
	})

	return err
}

// GetUnreadCount gets the count of unread notifications for a user
func (r *NotificationRepository) GetUnreadCount(userID primitive.ObjectID) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := r.notificationCollection.CountDocuments(ctx, bson.M{
		"userId": userID,
		"isRead": false,
	})

	return count, err
}
