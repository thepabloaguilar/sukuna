package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/thepabloaguilar/sukuna/core/sagas"
	"github.com/thepabloaguilar/sukuna/gateways/postgres"
	"github.com/thepabloaguilar/sukuna/gateways/step_execution"
	"log"
	"os"
	"os/signal"
)

const (
	consumerGroupName = "sukuna-worker"
	topicToConsume = "sukuna-out"
)

var (
	kafkaBrokers = []string{"localhost:9092"}
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt, os.Kill)
	defer func() {
		signal.Stop(interruptChannel)
		cancel()
	}()

	go func() {
		select {
		case <-interruptChannel:
			log.Println("interrupt signal received")
			cancel()
		case <-ctx.Done():
		}
		<-interruptChannel
		os.Exit(1)
	}()

	producer := createProducer()
	defer func() {
		if err := producer.Close(); err != nil {
			log.Fatalf("error closing the producer: %v", err)
		}
	}()

	databaseConnection, err := getDatabaseConnection(ctx)
	if err != nil {
		log.Fatalf("error getting db connection: %v", err)
	}
	database := postgres.New(databaseConnection)

	kafkaProducer := createProducer()

	stepExecutionGateway := step_execution.NewGateway(kafkaProducer)
	sagasRepository := postgres.NewSagaRepository(*database)
	sagaService := sagas.NewService(sagasRepository, stepExecutionGateway)

	consume(ctx, sagaService)

	select {
	case <-ctx.Done():
	}
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

func consume(ctx context.Context, sagaService sagas.Service) {
	consumerGroup, err := sarama.NewConsumerGroup(kafkaBrokers, consumerGroupName, nil)
	if err != nil {
		log.Fatalf("error creating a consumer group: %v", err)
	}

	consumer := Consumer{ctx: ctx, Ready: make(chan bool), SagaService: sagaService}
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("closing kafka connection")
				if err := consumerGroup.Close(); err != nil {
					log.Fatalf("error closing consumer group: %v", err)
				}

				return
			default:
				if err := consumerGroup.Consume(ctx, []string{topicToConsume}, &consumer); err != nil {
					log.Printf("error consuming: %v", err)
				}
			}
			consumer.Ready = make(chan bool)
		}
	}()

	// Waits for the consumer setup
	<-consumer.Ready
}

type SagaStepResult struct {
	SagaName string `json:"saga_name"`
	StepIndex int `json:"step_index"`
	ExecutionID uuid.UUID `json:"execution_id"`
	Result string `json:"result"`
}

type Consumer struct {
	ctx context.Context

	Ready chan bool
	SagaService sagas.Service
}

func (c *Consumer) Setup(_ sarama.ConsumerGroupSession) error {
	close(c.Ready)
	return nil
}

func (c Consumer) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (c Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		var result SagaStepResult
		err := json.Unmarshal(message.Value, &result)
		if err != nil {
			log.Printf("error unmarshaling result: %v\n", err)
		}

		log.Printf("result received: %v\n", result)
		vo := sagas.StepResultVO{
			SagaName:    result.SagaName,
			StepIndex:   result.StepIndex,
			ExecutionID: result.ExecutionID,
			Result:   result.Result,
		}
		if err := c.SagaService.HandleStepResult(c.ctx, vo); err != nil {
			log.Printf("error handling the result: %v", err)
		}

		session.MarkMessage(message, "")
	}
	return nil
}
