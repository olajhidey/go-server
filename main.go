package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type User struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	ProfileUrl string `json:"profileUrl"`
}

// Redis client setup
func redisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
}

// Router for Gin Web framework
func setUpRouter(redisClient *redis.Client) *gin.Engine {

	// context to run redis task in the background
	context := context.Background()

	r := gin.Default()
	r.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Endpoint to create new user
	r.POST("/user/create", func(ctx *gin.Context) {
		var user User
		if err := ctx.BindJSON(&user); err != nil {
			log.Fatal(err)
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		newUUID := uuid.NewString()

		jsonData, err := json.Marshal(user)

		if err != nil {
			fmt.Println("Error marshalling user:", err)
			return
		}

		err = redisClient.Set(context, string(newUUID), jsonData , 0).Err()

		if err != nil {
			fmt.Println("Error inserting user in Redis:", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})

			return
		}

		ctx.JSON(http.StatusCreated, gin.H{
			"sid":        string(newUUID),
			"name":       user.Name,
			"email":      user.Email,
			"profileUrl": user.ProfileUrl,
		})

	})

	// Retrieve a particular user based on ID
	r.GET("/user/:id", func(ctx *gin.Context) {

		var user User

		userId := ctx.Params.ByName("id")

		val, err := redisClient.Get(context, userId).Result()

		if err != nil {
			fmt.Println("User not found", userId)
			ctx.JSON(http.StatusNotFound, gin.H{
				"message": "User "+ userId +" Not found",
			})
			return
		}

		if err == redis.Nil {
			ctx.JSON(http.StatusNotFound, gin.H{
				"message": "User "+ userId +"Not found",
			})
		}else{	

			err := json.Unmarshal([]byte(val), &user)

			if err != nil {
				fmt.Println("Error unmarshalling user: ", err)
				return
			}

			ctx.JSON(http.StatusOK, gin.H{
				"sid": userId, 
				"data": user,
			})
		}

	})

	// Update a user information based on UserId 
	r.PUT("/user/update/:id", func(ctx *gin.Context) {
		userId := ctx.Params.ByName("id")

		var user User

		if err := ctx.BindJSON(&user); err != nil {
			log.Fatal(err)

			ctx.JSON(http.StatusOK, gin.H{
				"message": err.Error(),
			})

			return
		}
	
		userData, err := json.Marshal(user)

		if err != nil {
			fmt.Println("Error Marshalling user: ", user)
			return 
		}

		err = redisClient.Set(context, userId, userData, 0).Err()

		if err != nil {
			fmt.Println("Unable to update redis value")
		}

		ctx.JSON(http.StatusOK, user)
	})

	// Delete a user by ID
	r.DELETE("/user/:id", func(ctx *gin.Context) {
		userId := ctx.Params.ByName("id")

		_, err := redisClient.Del(context, userId).Result()

		if err != nil {
			fmt.Println("Error deleting key:", err)
			return
		}

		fmt.Println("User deleted: ", userId)

		ctx.JSON(http.StatusOK, gin.H{
			"message": "User removed successfully",
		})
	})

	return r

}

func main() {

	redisClient := redisClient()

	r := setUpRouter(redisClient)

	// App running at port 8080
	r.Run(":8080")
}
