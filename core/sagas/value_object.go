package sagas

import (
	"github.com/google/uuid"
	"github.com/thepabloaguilar/sukuna/core/entities"
)

type CreateSagaVO struct {
	Name    string
	Payload []byte
	Steps   []CreateSagaVOSteps
}

type CreateSagaVOSteps struct {
	Name string
}

type CreateSagaExecutionVO struct {
	SagaID  uuid.UUID
	Payload []byte
}

type SagaExecutionVO struct {
	entities.SagaExecution
	Steps []entities.StepExecution
}

type StepResultVO struct {
	SagaName string
	StepIndex int
	ExecutionID uuid.UUID
	Result string
}
