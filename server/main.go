package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"context"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var clientsLock = sync.Mutex{}
var messagesCollection *mongo.Collection

func main() {
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:5173"} // Replace with the origin of your React app
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE"}
	r.Use(cors.New(config))

	clientOptions := options.Client().ApplyURI("mongodb://mongo:5FqQMAHkTLMOGPp3P7GE@containers-us-west-39.railway.app:6667") // Update the connection string as needed
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Check the connection to MongoDB
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	// Create a MongoDB collection to store chat messages
	messagesCollection = client.Database("test").Collection("messages")

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Welcome to the chat app"})
	})
	// Add a new API endpoint to retrieve chat messages
	r.GET("/messages", func(c *gin.Context) {
		// Query MongoDB to retrieve chat messages
		messages, err := retrieveMessagesFromMongoDB()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve messages"})
			return
		}
		c.JSON(http.StatusOK, messages)
	})
	// Add a route to clear the MongoDB collection
	r.POST("/clear", func(c *gin.Context) {
		// Clear the MongoDB collection
		err := clearMessagesCollection()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear messages"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Messages collection cleared"})
	})

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close()

		// Add the new client to the map
		clientsLock.Lock()
		clients[conn] = true
		clientsLock.Unlock()

		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				log.Println(err)
				break
			}
			message := string(p)

			fmt.Printf("Received message: %s\n", message)

			err = saveMessageToMongoDB(message)
			if err != nil {
				log.Println(err)
			}

			// Broadcast the message to all connected clients
			broadcastMessage(messageType, p)
		}

		// Remove the client from the map when the connection is closed
		clientsLock.Lock()
		delete(clients, conn)
		clientsLock.Unlock()
	})

	r.Run(":8080")
}

func clearMessagesCollection() error {
	// Define a filter to match all documents (clear the entire collection)
	filter := bson.M{}

	// Delete all documents in the MongoDB collection
	_, err := messagesCollection.DeleteMany(context.TODO(), filter)
	if err != nil {
		return err
	}

	return nil
}

func retrieveMessagesFromMongoDB() ([]bson.M, error) {
	// Define a filter to query all chat messages
	filter := bson.M{}

	// Find chat messages in the MongoDB collection
	cur, err := messagesCollection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.TODO())

	// Create a slice to store the retrieved messages
	var messages []bson.M

	// Iterate through the cursor and decode messages
	for cur.Next(context.TODO()) {
		var message bson.M
		if err := cur.Decode(&message); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func saveMessageToMongoDB(message string) error {
	_, err := messagesCollection.InsertOne(context.TODO(), bson.M{"message": message})
	return err
}

func broadcastMessage(messageType int, message []byte) {
	clientsLock.Lock()
	defer clientsLock.Unlock()

	for client := range clients {
		err := client.WriteMessage(messageType, message)
		if err != nil {
			log.Println(err)
			client.Close()
			delete(clients, client)
		}
	}
}
