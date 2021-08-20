package entities

import (
	"time"

	"github.com/google/uuid"
)

type Saga struct {
	SagaID        uuid.UUID
	Name          string
	FormattedName string
	Payload       []byte
	CreatedAt     time.Time
}

type SagaStep struct {
	StepID uuid.UUID
	SagaID uuid.UUID
	Index  int
	Name   string
}

type SagaExecution struct {
	SagaExecutionID uuid.UUID
	SagaID          uuid.UUID
	Payload         []byte
	CreatedAt       time.Time
}

type StepExecution struct {
	StepExecutionID uuid.UUID
	SagaExecutionID uuid.UUID
	Index           int
	Name            string
	Status          StepExecutionStatus
}
