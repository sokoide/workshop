package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type User struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	LastSeen time.Time `json:"last_seen"`
}

func cacheKey(id string) string {
	return "user:" + id
}

func SetUser(ctx context.Context, client *redis.Client, user User, ttl time.Duration) error {
	payload, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return client.Set(ctx, cacheKey(user.ID), payload, ttl).Err()
}

func GetUser(ctx context.Context, client *redis.Client, id string) (User, bool, error) {
	value, err := client.Get(ctx, cacheKey(id)).Result()
	if err == redis.Nil {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}

	var user User
	if err := json.Unmarshal([]byte(value), &user); err != nil {
		return User{}, false, err
	}
	return user, true, nil
}

func main() {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	user := User{ID: "42", Name: "Ada", LastSeen: time.Now()}
	if err := SetUser(ctx, client, user, 30*time.Second); err != nil {
		fmt.Printf("set error: %v\n", err)
		return
	}

	cached, ok, err := GetUser(ctx, client, "42")
	if err != nil {
		fmt.Printf("get error: %v\n", err)
		return
	}
	if ok {
		fmt.Printf("cache hit: %+v\n", cached)
	}
}
