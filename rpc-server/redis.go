package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	cli *redis.Client
}

func (c *RedisClient) InitClient(ctx context.Context, address, password string) error {
	r := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password, // no password set
		DB:       0,        // use default DB
	})

	if err := r.Ping(ctx).Err(); err != nil {
		return err
	}

	c.cli = r
	return nil
}

func (c *RedisClient) SaveMessageToRedis(ctx context.Context, key string, message *Message) error {
	// Marshal the message struct to JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Save the message to Redis using the key and JSON value
	err = c.cli.HSet(ctx, key, message.Timestamp, messageJSON).Err()
	if err != nil {
		return err
	}

	return nil
}

func (c *RedisClient) GetMessagesByGroupID(ctx context.Context, groupID string, start, end int64, reverse bool) ([]*Message, error) {
	var (
		rawMessages map[string]string
		messages    []*Message
		err         error
	)

	fmt.Printf("Retrieving messages from Redis with key %s\n", groupID)

	// Retrieve all fields and values of the hash
	rawMessages, err = c.cli.HGetAll(ctx, groupID).Result()
	if err != nil {
		return nil, err
	}

	// Create a slice to store the messages
	messageSlice := make([]*Message, 0, len(rawMessages))

	// Iterate over the raw messages and append them to the slice
	for _, msgJSON := range rawMessages {
		temp := &Message{}
		err := json.Unmarshal([]byte(msgJSON), temp)
		if err != nil {
			return nil, err
		}
		messageSlice = append(messageSlice, temp)
	}

	// Sort the messages based on the timestamp
	if reverse {
		sort.SliceStable(messageSlice, func(i, j int) bool {
			return messageSlice[i].Timestamp > messageSlice[j].Timestamp
		})
	} else {
		sort.SliceStable(messageSlice, func(i, j int) bool {
			return messageSlice[i].Timestamp < messageSlice[j].Timestamp
		})
	}

	// Apply start and end indices
	startIndex := int(start)
	endIndex := int(end)
	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex >= len(messageSlice) {
		endIndex = len(messageSlice) - 1
	}
	if endIndex < startIndex {
		return nil, fmt.Errorf("invalid start and end indices")
	}

	// Retrieve the selected range of messages
	messages = messageSlice[startIndex : endIndex+1]

	fmt.Println("Messages retrieved from Redis:", messages)

	return messages, nil
}
