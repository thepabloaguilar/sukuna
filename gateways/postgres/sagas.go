package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/thepabloaguilar/sukuna/core/entities"
)

type SagaRepository struct {
	q Queries
}

func NewSagaRepository(q Queries) SagaRepository {
	return SagaRepository{q: q}
}

func (r SagaRepository) GetSaga(ctx context.Context, sagaID uuid.UUID) (entities.Saga, error) {
	dbSaga, err := r.q.GetSaga(ctx, sagaID)
	if err != nil {
		return entities.Saga{}, err
	}

	return entities.Saga{
		SagaID:        dbSaga.SagaID,
		Name:          dbSaga.Name,
		FormattedName: dbSaga.FormattedName,
		Payload:       dbSaga.Payload,
		CreatedAt:     dbSaga.CreatedAt,
	}, nil
}

func (r SagaRepository) CreateSaga(ctx context.Context, saga entities.Saga) (entities.Saga, error) {
	args := CreateSagaParams{
		Name:          saga.Name,
		FormattedName: saga.FormattedName,
		Payload:       saga.Payload,
	}
	dbSaga, err := r.q.CreateSaga(ctx, args)
	if err != nil {
		return entities.Saga{}, err
	}

	return entities.Saga{
		SagaID:        dbSaga.SagaID,
		Name:          dbSaga.Name,
		FormattedName: dbSaga.FormattedName,
		Payload:       dbSaga.Payload,
		CreatedAt:     dbSaga.CreatedAt,
	}, nil
}

func (r SagaRepository) GetSagaStepsBySagaID(
	ctx context.Context,
	sagaID uuid.UUID,
) ([]entities.SagaStep, error) {
	dbSteps, err := r.q.GetSagaStepsBySagaID(ctx, sagaID)
	if err != nil {
		return nil, fmt.Errorf("error getting saga steps: %w", err)
	}

	sagaSteps := make([]entities.SagaStep, 0, len(dbSteps))
	for _, step := range dbSteps {
		sagaStep := entities.SagaStep{
			StepID: step.StepID,
			SagaID: step.SagaID,
			Index:  int(step.Index),
			Name:   step.Name,
		}
		sagaSteps = append(sagaSteps, sagaStep)
	}

	return sagaSteps, nil
}

func (r SagaRepository) CreateSagaSteps(
	ctx context.Context,
	steps []entities.SagaStep,
) ([]entities.SagaStep, error) {
	args := CreateSagaStepsParams{
		SagaIds: make([]uuid.UUID, 0, len(steps)),
		Indexes: make([]int32, 0, len(steps)),
		Names:   make([]string, 0, len(steps)),
	}

	for _, step := range steps {
		args.SagaIds = append(args.SagaIds, step.SagaID)
		args.Indexes = append(args.Indexes, int32(step.Index))
		args.Names = append(args.Names, step.Name)
	}

	dbSteps, err := r.q.CreateSagaSteps(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("error saving saga steps: %w", err)
	}

	savedSteps := make([]entities.SagaStep, 0, len(steps))
	for _, step := range dbSteps {
		savedSteps = append(savedSteps, entities.SagaStep{
			StepID: step.StepID,
			SagaID: step.SagaID,
			Index:  int(step.Index),
			Name:   step.Name,
		})
	}

	return savedSteps, nil
}

func (r SagaRepository) GetSagaExecution(
	ctx context.Context,
	executionID uuid.UUID,
) (entities.SagaExecution, error) {
	dbExecution, err := r.q.GetSagaExecution(ctx, executionID)
	if err != nil {
		return entities.SagaExecution{}, fmt.Errorf("error getting saga execution info: %w", err)
	}

	return entities.SagaExecution{
		SagaExecutionID: dbExecution.SagaExecutionID,
		SagaID:          dbExecution.SagaID,
		Payload:         dbExecution.Payload,
		CreatedAt:       dbExecution.CreatedAt,
	}, nil
}

func (r SagaRepository) CreateSagaExecution(
	ctx context.Context,
	execution entities.SagaExecution,
) (entities.SagaExecution, error) {
	args := CreateSagaExecutionParams{
		SagaID:  execution.SagaID,
		Payload: execution.Payload,
	}
	savedExecution, err := r.q.CreateSagaExecution(ctx, args)
	if err != nil {
		return entities.SagaExecution{}, fmt.Errorf("error saving saga execution: %w", err)
	}

	return entities.SagaExecution{
		SagaExecutionID: savedExecution.SagaExecutionID,
		SagaID:          savedExecution.SagaID,
		Payload:         savedExecution.Payload,
		CreatedAt:       savedExecution.CreatedAt,
	}, err
}

func (r SagaRepository) GetSagaStepsExecutionByExecutionID(
	ctx context.Context,
	executionID uuid.UUID,
) ([]entities.StepExecution, error) {
	executions, err := r.q.GetSagaStepsExecutionByExecutionID(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("error getting steps execution info: %w", err)
	}

	stepExecutions := make([]entities.StepExecution, 0, len(executions))
	for _, execution := range executions {
		stepExecution := entities.StepExecution{
			StepExecutionID: execution.StepExecutionID,
			SagaExecutionID: execution.SagaExecutionID,
			Index:           int(execution.Index),
			Name:            execution.Name,
			Status:          entities.StepExecutionStatus(execution.Status),
		}
		stepExecutions = append(stepExecutions, stepExecution)
	}

	return stepExecutions, nil
}

func (r SagaRepository) CreateSagaStepsExecution(
	ctx context.Context,
	steps []entities.StepExecution,
) ([]entities.StepExecution, error) {
	args := CreateSagaStepsExecutionParams{
		SagaExecutionIds: make([]uuid.UUID, 0, len(steps)),
		Indexes:          make([]int32, 0, len(steps)),
		Names:            make([]string, 0, len(steps)),
		Statuses:         make([]string, 0, len(steps)),
	}

	for _, step := range steps {
		args.SagaExecutionIds = append(args.SagaExecutionIds, step.SagaExecutionID)
		args.Indexes = append(args.Indexes, int32(step.Index))
		args.Names = append(args.Names, step.Name)
		args.Statuses = append(args.Statuses, string(step.Status))
	}
	savedSteps, err := r.q.CreateSagaStepsExecution(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("error saving steps execution: %w", err)
	}

	stepsExecution := make([]entities.StepExecution, 0, len(savedSteps))
	for _, step := range savedSteps {
		stepExecution := entities.StepExecution{
			StepExecutionID: step.StepExecutionID,
			SagaExecutionID: step.SagaExecutionID,
			Index:           int(step.Index),
			Name:            step.Name,
			Status:          entities.StepExecutionStatus(step.Status),
		}
		stepsExecution = append(stepsExecution, stepExecution)
	}

	return stepsExecution, nil
}

func (r SagaRepository) SetSagaStepExecutionStatus(
	ctx context.Context,
	status entities.StepExecutionStatus,
	index int,
	executionID uuid.UUID,
) error {
	params := SetSagaStepExecutionStatusParams{
		Status:          string(status),
		Index: int32(index),
		SagaExecutionID: executionID,
	}
	return r.q.SetSagaStepExecutionStatus(ctx, params)
}
