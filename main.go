package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"fmt"
	"math/rand"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Product struct {
	ID          primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Title       string             `json:"title,omitempty" bson:"title,omitempty"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Image       string             `json:"image,omitempty" bson:"image,omitempty"`
	Price       int                `json:"price,omitempty" bson:"price,omitempty"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	env := os.Getenv("MONGO_URI")

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, _ := mongo.Connect(ctx, options.Client().ApplyURI(env))
	db := client.Database("go_search")

	app := fiber.New()

	app.Use(cors.New())

	app.Post("/api/products/populate", func(c *fiber.Ctx) error {
		collection := db.Collection("products")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

		for i := 0; i < 50; i++ {
			collection.InsertOne(ctx, Product{
				Title:       faker.Word(),
				Description: faker.Paragraph(),
				Image:       fmt.Sprintf("http://lorempixel.com/200/200?%s", faker.UUIDDigit()),
				Price:       rand.Intn(90) + 10,
			})
		}
		return c.JSON(fiber.Map{
			"message": "success",
		})

	})

	app.Get("/api/products/frontend", func(c *fiber.Ctx) error {
		collection := db.Collection("products")
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		var products []Product
		cursor, _ := collection.Find(ctx, fiber.Map{})
		defer cursor.Close(ctx)

		for cursor.Next(ctx) {
			var product Product
			cursor.Decode(&product)
			products = append(products, product)
		}

		return c.JSON(products)
	})
	app.Get("/api/products/backend", func(c *fiber.Ctx) error {
		collection := db.Collection("products")
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		var products []Product

		filter := bson.M{}
		findOptions := options.Find()

		if s := c.Query("s"); s != "" {
			filter = bson.M{
				"$or": []bson.M{
					{
						"title": bson.M{
							"$regex": primitive.Regex{
								Pattern: s,
								Options: "i",
							},
						},
					},
					{
						"description": bson.M{
							"$regex": primitive.Regex{
								Pattern: s,
								Options: "i",
							},
						},
					},
				},
			}
		}

		if sort := c.Query("sort"); sort != "" {
			if sort == "asc" {
				findOptions.SetSort(bson.D{{"price", 1}})
			} else if sort == "desc" {
				findOptions.SetSort(bson.D{{"price", -1}})
			}
		}

		page, _ := strconv.Atoi(c.Query("page", "1"))
		var perPage int64 = 10

		total, _ := collection.CountDocuments(ctx, filter)
		count, _ := collection.CountDocuments(ctx, filter, &options.CountOptions{
			Limit: &perPage,
		})

		findOptions.SetSkip((int64(page) - 1) * perPage)
		findOptions.SetLimit(perPage)

		cursor, _ := collection.Find(ctx, filter, findOptions)
		defer cursor.Close(ctx)

		for cursor.Next(ctx) {
			var product Product
			cursor.Decode(&product)
			products = append(products, product)
		}

		return c.JSON(fiber.Map{
			"total_items":  total,
			"item_count":   count,
			"current_page": page,
			"total_page":   int64(total) / perPage,
			"data":         products,
		})
	})

	app.Listen(":8000")
}
