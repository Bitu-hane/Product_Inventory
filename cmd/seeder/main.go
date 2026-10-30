package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

type Product struct {
	name           string
	sku            string
	description    string
	price          string
	stock_quantity int
	status         string
}

func main() {
	product := []Product{
		{
			name:           "HP",
			sku:            "ABC123",
			description:    "laptop ",
			price:          "1150.00",
			stock_quantity: 0,
			status:         "active",
		},
		{
			name:           "Lenovo",
			sku:            "FGH456",
			description:    "Laptop",
			price:          "1199.99",
			stock_quantity: 10,
			status:         "active",
		},
		{
			name:           "MAC BOOK Pro",
			sku:            "RTY789",
			description:    "Laptop",
			price:          "100.00",
			stock_quantity: 18,
			status:         "draft",
		},
		{
			name:           "Dell",
			sku:            "IJK456",
			description:    "Laptop",
			price:          "950.00",
			stock_quantity: 14,
			status:         "draft",
		},
		{
			name:           "Microsoft Surface",
			sku:            "QWE123",
			description:    "Laptop",
			price:          "1299.99",
			stock_quantity: 2,
			status:         "archived",
		},
		{
			name:           "Samsung",
			sku:            "SKU456",
			description:    "Andriod phone",
			price:          "199.99",
			stock_quantity: 24,
			status:         "active",
		},
		{
			name:           "iPhone 18 pro max",
			sku:            "CDT999",
			description:    "Phone",
			price:          "2499.99",
			stock_quantity: 14,
			status:         "draft",
		},
		{
			name:           "Huwaei P60",
			sku:            "LKH123",
			description:    "Phone",
			price:          "1599.99",
			stock_quantity: 24,
			status:         "archived",
		},
		{
			name:           "Itel",
			sku:            "LLL444",
			description:    "Phone",
			price:          "499.99",
			stock_quantity: 4,
			status:         "active",
		},
		{
			name:           "Tecno",
			sku:            "TTT456",
			description:    "Phone",
			price:          "499.99",
			stock_quantity: 4,
			status:         "active",
		},
	}

	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DB_URL")
	}
	if dbURL == "" {
		dbURL = "postgres://postgres@localhost:5432/product_inventory?sslmode=disable"
	}

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx)

	query := `INSERT INTO products(name, sku, description, price, stock_quantity, status) 
	          VALUES($1, $2, $3, $4, $5, $6) 
	          ON CONFLICT (sku) DO NOTHING`

	for i := 0; i < len(product); i++ {
		p := product[i]
		tag, err := conn.Exec(ctx, query, p.name, p.sku, p.description, p.price, p.stock_quantity, p.status)
		if err != nil {
			log.Fatalf("Failed to insert product %s (SKU: %s): %v\n", p.name, p.sku, err)
		}

		if tag.RowsAffected() > 0 {
			fmt.Printf("Inserted: %s (SKU: %s)\n", p.name, p.sku)
		} else {
			fmt.Printf("Skipped (already exists): %s (SKU: %s)\n", p.name, p.sku)
		}
	}

	fmt.Println("Seeding completed successfully!")
}
