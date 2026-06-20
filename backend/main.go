package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"certpulse/backend/db"
	"certpulse/backend/scanner"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/google/uuid"
)

func main() {
	// Initialize database
	db.Connect()
	defer db.Close()

	app := fiber.New(fiber.Config{
		AppName: "CertPulse API Service",
	})

	// Add Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	// API Routes Group
	api := app.Group("/api")

	api.Get("/status", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Get all monitored endpoints
	api.Get("/endpoints", func(c *fiber.Ctx) error {
		workspaceID := c.Query("workspace_id")
		if workspaceID == "" {
			// Fallback to default developer workspace
			workspaceID = "b27e69f8-b3d9-43c2-84bb-762bc2b55f24"
		}

		rows, err := db.Pool.Query(context.Background(), `
			SELECT e.id, e.domain_name, e.port, e.last_scan_status, e.last_scan_at,
			       c.common_name, c.valid_to, c.chain_valid, c.issuer_organization
			FROM monitored_endpoints e
			LEFT JOIN certificates c ON e.active_certificate_id = c.id
			WHERE e.workspace_id = $1
			ORDER BY e.created_at DESC
		`, workspaceID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		endpoints := []fiber.Map{}
		for rows.Next() {
			var id, domain, status string
			var port int
			var lastScan *time.Time
			var commonName, issuerOrg *string
			var validTo *time.Time
			var chainValid *bool

			err := rows.Scan(&id, &domain, &port, &status, &lastScan, &commonName, &validTo, &chainValid, &issuerOrg)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}

			endpoints = append(endpoints, fiber.Map{
				"id":                  id,
				"domain_name":         domain,
				"port":                port,
				"last_scan_status":    status,
				"last_scan_at":        lastScan,
				"common_name":         commonName,
				"valid_to":            validTo,
				"chain_valid":         chainValid,
				"issuer_organization": issuerOrg,
			})
		}

		return c.JSON(endpoints)
	})

	// Add new domain to monitor
	api.Post("/endpoints", func(c *fiber.Ctx) error {
		type Request struct {
			DomainName  string `json:"domain_name"`
			Port        int    `json:"port"`
			WorkspaceID string `json:"workspace_id"`
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}

		if req.DomainName == "" {
			return c.Status(400).JSON(fiber.Map{"error": "domain_name is required"})
		}

		if req.Port == 0 {
			req.Port = 443
		}

		if req.WorkspaceID == "" {
			req.WorkspaceID = "b27e69f8-b3d9-43c2-84bb-762bc2b55f24"
		}

		var id string
		err := db.Pool.QueryRow(context.Background(), `
			INSERT INTO monitored_endpoints (workspace_id, domain_name, port, last_scan_status)
			VALUES ($1, $2, $3, 'pending')
			RETURNING id
		`, req.WorkspaceID, req.DomainName, req.Port).Scan(&id)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Trigger an initial scan asynchronously
		go func(endpointID string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = scanner.TriggerScan(ctx, endpointID)
		}(id)

		return c.Status(201).JSON(fiber.Map{
			"message":     "Endpoint added successfully",
			"endpoint_id": id,
		})
	})

	// Force scan an endpoint
	api.Post("/endpoints/:id/scan", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Endpoint ID is required"})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		err := scanner.TriggerScan(ctx, id)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   fmt.Sprintf("Scan failed: %v", err),
				"status":  "failed",
			})
		}

		return c.JSON(fiber.Map{
			"message": "Scan executed successfully",
			"status":  "success",
		})
	})

	// Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Printf("Starting CertPulse backend API on port %s...\n", port)
	log.Fatal(app.Listen(fmt.Sprintf(":%s", port)))
}
