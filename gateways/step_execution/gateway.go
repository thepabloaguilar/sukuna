package step_execution

import (
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/google/uuid"
	"github.com/thepabloaguilar/sukuna/core/entities"
	"github.com/thepabloaguilar/sukuna/core/sagas"
)

type StepToExecute struct {
	SagaName string `json:"saga_name"`
	StepIndex int `json:"step_index"`
	ExecutionID uuid.UUID `json:"saga_id"`
	Payload json.RawMessage `json:"payload"`
	IsCompensation bool `json:"is_compensation"`
}

type gateway struct {
	producer sarama.SyncProducer
}

func NewGateway(producer sarama.SyncProducer) sagas.StepExecutionGateway {
	return gateway{producer: producer}
}

func (g gateway) SendStepToExecute(
	sagaName string,
	sagaExecution entities.SagaExecution,
	sagaStep entities.StepExecution,
	isCompensation bool,
) error {
	value := StepToExecute{
		SagaName: sagaName,
		StepIndex: sagaStep.Index,
		ExecutionID: sagaExecution.SagaExecutionID,
		Payload: sagaExecution.Payload,
		IsCompensation: isCompensation,
	}

	marshaledValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("error marsheling the step: %w", err)
	}

	_, _, err = g.producer.SendMessage(&sarama.ProducerMessage{
		Topic:     fmt.Sprintf("%s-%s", sagaName, sagaStep.Name),
		Value:     sarama.ByteEncoder(marshaledValue),
	})
	if err != nil {
		return fmt.Errorf("error sending step to be executed: %w", err)
	}

	return nil
}
