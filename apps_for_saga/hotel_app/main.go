package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"

	"github.com/Shopify/sarama"
	"github.com/google/uuid"
	"github.com/thepabloaguilar/sukuna/gateways/step_execution"
)

const (
	consumerGroupName = "hotel-worker"
	topicToConsume    = "trip-saga-hotel-step"
	resultTopic       = "sukuna-out"
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

	consume(ctx, producer)

	select {
	case <-ctx.Done():
	}
}

func createProducer() sarama.SyncProducer {
	producer, err := sarama.NewSyncProducer(kafkaBrokers, nil)
	if err != nil {
		log.Fatalf("error creating the producer: %v", err)
	}

	return producer
}

func consume(ctx context.Context, producer sarama.SyncProducer) {
	consumerGroup, err := sarama.NewConsumerGroup(kafkaBrokers, consumerGroupName, nil)
	if err != nil {
		log.Fatalf("error creating a consumer group: %v", err)
	}

	consumer := Consumer{Ready: make(chan bool), Producer: producer}
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

type Payload struct {
	PaymentAmount     float64 `json:"payment_amount"`
	HotelName         string  `json:"hotel_name"`
	FlightCompanyName string  `json:"flight_company_name"`
}

type SagaStepResult struct {
	SagaName    string    `json:"saga_name"`
	StepIndex   int       `json:"step_index"`
	ExecutionID uuid.UUID `json:"execution_id"`
	Result string `json:"result"`
}

type Consumer struct {
	Ready    chan bool
	Producer sarama.SyncProducer
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
		step, _ := messageValueToStepToExecute(message.Value)
		log.Printf("Saga Execution received: %s\n", step.ExecutionID)

		if step.IsCompensation {
			log.Println("compensated")

			if err := c.sendCompensated(step); err != nil {
				log.Printf("error compensanting: %v", err)
			}
			continue
		}

		var payload Payload
		if err := json.Unmarshal(step.Payload, &payload); err != nil {
			log.Printf("error unmarshaling payload: %v", err)
			continue
		}

		if payload.HotelName == "HOTEL XABLAUZER" {
			log.Println("Sending success")

			err := c.sendSuccess(step)
			if err != nil {
				log.Printf("An error occurred: %v\n", err)
			}
		} else {
			log.Println("Sending failure")

			err := c.sendError(step)
			if err != nil {
				log.Printf("An error occurred: %v\n", err)
			}
		}

		session.MarkMessage(message, "")
	}
	return nil
}

func (c Consumer) sendSuccess(step step_execution.StepToExecute) error {
	result := SagaStepResult{
		SagaName:    step.SagaName,
		StepIndex:   step.StepIndex,
		ExecutionID: step.ExecutionID,
		Result:      "success",
	}
	return c.sendResult(result)
}

func (c Consumer) sendError(step step_execution.StepToExecute) error {
	result := SagaStepResult{
		SagaName:    step.SagaName,
		StepIndex:   step.StepIndex,
		ExecutionID: step.ExecutionID,
		Result:      "error",
	}
	return c.sendResult(result)
}

func (c Consumer) sendCompensated(step step_execution.StepToExecute) error {
	result := SagaStepResult{
		SagaName:    step.SagaName,
		StepIndex:   step.StepIndex,
		ExecutionID: step.ExecutionID,
		Result:      "compensated",
	}
	return c.sendResult(result)
}

func (c Consumer) sendResult(result SagaStepResult) error {
	marshaledResult, err := json.Marshal(result)
	if err != nil {
		return err
	}

	_, _, err = c.Producer.SendMessage(&sarama.ProducerMessage{
		Topic: resultTopic,
		Value: sarama.ByteEncoder(marshaledResult),
	})

	return err
}

func messageValueToStepToExecute(value []byte) (step_execution.StepToExecute, error) {
	var stepToExecute step_execution.StepToExecute
	err := json.Unmarshal(value, &stepToExecute)
	return stepToExecute, err
}
