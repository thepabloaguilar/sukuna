package sagas

import (
	"context"
	"github.com/google/uuid"
	"github.com/thepabloaguilar/sukuna/core/entities"
)

type SagaRepository interface {
	// Sagas

	GetSaga(ctx context.Context, sagaID uuid.UUID) (entities.Saga, error)
	CreateSaga(ctx context.Context, saga entities.Saga) (entities.Saga, error)

	// Saga Steps

	GetSagaStepsBySagaID(ctx context.Context, sagaID uuid.UUID) ([]entities.SagaStep, error)
	CreateSagaSteps(ctx context.Context, steps []entities.SagaStep) ([]entities.SagaStep, error)

	// Saga Execution

	GetSagaExecution(ctx context.Context, executionID uuid.UUID) (entities.SagaExecution, error)
	CreateSagaExecution(ctx context.Context, execution entities.SagaExecution) (entities.SagaExecution, error)

	// Saga Steps Execution

	GetSagaStepsExecutionByExecutionID(ctx context.Context, executionID uuid.UUID) ([]entities.StepExecution, error)
	CreateSagaStepsExecution(ctx context.Context, steps []entities.StepExecution) ([]entities.StepExecution, error)
	SetSagaStepExecutionStatus(ctx context.Context, status entities.StepExecutionStatus, index int, executionID uuid.UUID) error
}

type StepExecutionGateway interface {
	SendStepToExecute(
		sagaName string,
		sagaExecution entities.SagaExecution,
		sagaStep entities.StepExecution,
		isCompensation bool,
	) error
}
