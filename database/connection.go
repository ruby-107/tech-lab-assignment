package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func ConnectDB() {

	var err error
	for i := 0; i < 5; i++ {
		DB, err = sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/employee")
		if err == nil {
			if err = DB.Ping(); err == nil {
				log.Println("Database connected successfully")
				return
			}
		}
		log.Printf("Failed to connect to database: %v. Retrying...\n", err)
		time.Sleep(2 * time.Second)
	}
	log.Fatal("Could not connect to the database after several attempts")

	// Create table if not exists
	createTable := `CREATE TABLE IF NOT EXISTS employees (
        id INT AUTO_INCREMENT,
        first_name VARCHAR(50),
        last_name VARCHAR(50),
        gender VARCHAR(10),
        country VARCHAR(50),
        age INT,
        date VARCHAR(20)
    );`
	if _, err := DB.Exec(createTable); err != nil {
		panic(err)
	}

	fmt.Println("Database connected and table created")
}

var RedisClient *redis.Client
var RedisCtx = context.Background()

func ConnectRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Use the service name defined in docker-compose
		Password: "",               // No password set
		DB:       0,                // Use default DB
	})

	// Test the connection with retry
	for i := 0; i < 5; i++ {
		_, err := RedisClient.Ping(RedisCtx).Result()
		if err == nil {
			log.Println("Connected to Redis!")
			return
		}

		log.Println("Could not connect to Redis, retrying...")
		time.Sleep(2 * time.Second)
	}
}
