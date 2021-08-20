package main

import (
	"context"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/thepabloaguilar/sukuna/gateways/step_execution"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/thepabloaguilar/sukuna/core/sagas"
	"github.com/thepabloaguilar/sukuna/entrypoints/api/routes"
	"github.com/thepabloaguilar/sukuna/gateways/postgres"
)

var kafkaBrokers = []string{"localhost:9092"}

func main() {
	if err := run(); err != nil {
		log.Fatalf("error running the application: %v", err)
	}
}

func run() error {
	app := fiber.New()
	ctx := context.Background()

	databaseConnection, err := getDatabaseConnection(ctx)
	if err != nil {
		return err
	}
	database := postgres.New(databaseConnection)

	kafkaProducer := createProducer()

	stepExecutionGateway := step_execution.NewGateway(kafkaProducer)
	sagasRepository := postgres.NewSagaRepository(*database)
	sagaService := sagas.NewService(sagasRepository, stepExecutionGateway)

	registerApiV1Routes(app, sagaService)

	if err := app.Listen(":8080"); err != nil {
		return fmt.Errorf("error starting server: %w", err)
	}

	return nil
}

func registerApiV1Routes(app *fiber.App, sagaService sagas.Service) {
	api := app.Group("/api/v1")

	routes.SagaRouter(api, sagaService)
}

func getDatabaseConnection(ctx context.Context) (*pgxpool.Pool, error) {
	databaseUrl := "postgres://sukuna:sukuna@localhost:5432/sukuna"
	dbConfig, err := pgxpool.ParseConfig(databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("error parsing db config: %w", err)
	}

	dbConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		conn.ConnInfo().RegisterDataType(pgtype.DataType{
			Value: &pgtype.UUID{},
			Name:  "uuid",
			OID:   pgtype.UUIDOID,
		})
		return nil
	}

	connection, err := pgxpool.ConnectConfig(ctx, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating db connection pool: %w", err)
	}

	return connection, nil
}

func createProducer() sarama.SyncProducer {
	producer, err := sarama.NewSyncProducer(kafkaBrokers, nil)
	if err != nil {
		log.Fatalf("error creating the producer: %v", err)
	}

	return producer
}
