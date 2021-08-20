package sagas

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/qri-io/jsonschema"
	"github.com/thepabloaguilar/sukuna/core/entities"
)

var ErrInvalidJSONSchema = errors.New("invalid json schema")
var ErrSagaNotFound = errors.New("saga not found")

type Service interface {
	CreateSaga(ctx context.Context, vo CreateSagaVO) (entities.Saga, error)
	GetSagaByID(ctx context.Context, sagaID uuid.UUID) (entities.Saga, error)
	CreateSagaExecution(ctx context.Context, vo CreateSagaExecutionVO) (entities.SagaExecution, error)
	GetSagaExecution(ctx context.Context, executionID uuid.UUID) (SagaExecutionVO, error)
	HandleStepResult(ctx context.Context, result StepResultVO) error
}

type service struct {
	repository       SagaRepository
	executionGateway StepExecutionGateway
}

func NewService(repository SagaRepository, executionGateway StepExecutionGateway) Service {
	return service{
		repository:       repository,
		executionGateway: executionGateway,
	}
}

func (svc service) CreateSaga(
	ctx context.Context,
	vo CreateSagaVO,
) (entities.Saga, error) {
	if err := svc.validateJSONSchema(vo.Payload); err != nil {
		return entities.Saga{}, err
	}

	saga := entities.Saga{
		Name:          vo.Name,
		FormattedName: svc.formatSagaName(vo.Name),
		Payload:       vo.Payload,
	}
	savedSaga, err := svc.repository.CreateSaga(ctx, saga)
	if err != nil {
		return entities.Saga{}, fmt.Errorf("error saving saga: %w", err)
	}

	sagaSteps := make([]entities.SagaStep, 0, len(vo.Steps))
	for index, voStep := range vo.Steps {
		step := entities.SagaStep{
			SagaID: savedSaga.SagaID,
			Index:  index + 1,
			// TODO: Create `formatted_name` attr
			Name: svc.formatSagaName(voStep.Name),
		}
		sagaSteps = append(sagaSteps, step)
	}
	_, err = svc.repository.CreateSagaSteps(ctx, sagaSteps)
	if err != nil {
		return entities.Saga{}, fmt.Errorf("error saving saga steps: %w", err)
	}

	return savedSaga, nil
}

func (svc service) formatSagaName(name string) string {
	return strings.ToLower(
		strings.ReplaceAll(
			strings.ReplaceAll(name, " ", "-"), "_", "-",
		),
	)
}

func (svc service) validateJSONSchema(payload []byte) error {
	jsonSchema := jsonschema.Schema{}
	if err := jsonSchema.UnmarshalJSON(payload); err != nil {
		return ErrInvalidJSONSchema
	}

	return nil
}

func (svc service) GetSagaByID(
	ctx context.Context,
	sagaID uuid.UUID,
) (entities.Saga, error) {
	return svc.repository.GetSaga(ctx, sagaID)
}

func (svc service) CreateSagaExecution(
	ctx context.Context,
	vo CreateSagaExecutionVO,
) (entities.SagaExecution, error) {
	saga, err := svc.repository.GetSaga(ctx, vo.SagaID)
	if err != nil {
		return entities.SagaExecution{}, err
	}
	sagaSteps, err := svc.repository.GetSagaStepsBySagaID(ctx, saga.SagaID)
	if err != nil {
		return entities.SagaExecution{}, err
	}

	err = svc.validatePayload(ctx, saga.Payload, vo.Payload)
	if err != nil {
		return entities.SagaExecution{}, err
	}

	sagaExecution := entities.SagaExecution{
		SagaID:  saga.SagaID,
		Payload: vo.Payload,
	}
	savedExecution, err := svc.repository.CreateSagaExecution(ctx, sagaExecution)
	if err != nil {
		return entities.SagaExecution{}, fmt.Errorf("error saving saga execution: %w", err)
	}

	sagaExecutionSteps := make([]entities.StepExecution, 0, len(sagaSteps))
	for _, step := range sagaSteps {
		stepExecution := entities.StepExecution{
			SagaExecutionID: savedExecution.SagaExecutionID,
			Index:           step.Index,
			Name:            step.Name,
			Status:          entities.StepExecutionRegistered,
		}
		sagaExecutionSteps = append(sagaExecutionSteps, stepExecution)
	}
	_, err = svc.repository.CreateSagaStepsExecution(ctx, sagaExecutionSteps)
	if err != nil {
		return entities.SagaExecution{}, fmt.Errorf("error saving saga step execution: %w", err)
	}

	firstStep := sagaExecutionSteps[0]
	err = svc.repository.SetSagaStepExecutionStatus(
		ctx, entities.StepExecutionStarted, firstStep.Index, firstStep.SagaExecutionID,
	)
	if err != nil {
		return entities.SagaExecution{}, err
	}

	err = svc.executionGateway.SendStepToExecute(saga.FormattedName, savedExecution, sagaExecutionSteps[0], false)
	if err != nil {
		return entities.SagaExecution{}, fmt.Errorf("error sending the first step to be executed: %w", err)
	}

	return savedExecution, nil
}

func (svc service) validatePayload(ctx context.Context, schema []byte, payload []byte) error {
	jsonSchema := jsonschema.Schema{}
	if err := jsonSchema.UnmarshalJSON(schema); err != nil {
		return ErrInvalidJSONSchema
	}

	validationErrors, err := jsonSchema.ValidateBytes(ctx, payload)
	if err != nil {
		return fmt.Errorf("error validating payload: %w", err)
	}
	if len(validationErrors) > 0 {
		return fmt.Errorf("payload error: %s", validationErrors[0].Message)
	}

	return nil
}

func (svc service) GetSagaExecution(ctx context.Context, executionID uuid.UUID) (SagaExecutionVO, error) {
	sagaExecution, err := svc.repository.GetSagaExecution(ctx, executionID)
	if err != nil {
		return SagaExecutionVO{}, err
	}

	stepsExecution, err := svc.repository.GetSagaStepsExecutionByExecutionID(ctx, executionID)
	if err != nil {
		return SagaExecutionVO{}, err
	}

	vo := SagaExecutionVO{
		SagaExecution: sagaExecution,
		Steps:         stepsExecution,
	}
	return vo, nil
}

func (svc service) HandleStepResult(ctx context.Context, result StepResultVO) error {
	switch result.Result {
	case "success":
		return svc.onSuccessResult(ctx, result)
	case "error":
		return svc.onFailureResult(ctx, result)
	case "compensated":
		return svc.onCompensation(ctx, result)
	default:
		return nil
	}
}

func (svc service) onSuccessResult(ctx context.Context, result StepResultVO) error {
	// Get Saga Execution
	sagaExecution, err := svc.repository.GetSagaExecution(ctx, result.ExecutionID)
	if err != nil {
		return err
	}

	// Get Step Execution
	stepsExecution, err := svc.repository.GetSagaStepsExecutionByExecutionID(ctx, result.ExecutionID)
	if err != nil {
		return err
	}

	// Mark the received step result as finished
	err = svc.repository.SetSagaStepExecutionStatus(
		ctx, entities.StepExecutionFinished, result.StepIndex, result.ExecutionID,
	)
	if err != nil {
		return err
	}

	// Get the next step and mark it as started
	nextStep := findNextStep(result.StepIndex, stepsExecution)
	if nextStep == nil {
		// Make something better later
		log.Println("Saga was finished")
		return nil
	}

	err = svc.repository.SetSagaStepExecutionStatus(
		ctx, entities.StepExecutionStarted, nextStep.Index, result.ExecutionID,
	)
	if err != nil {
		return err
	}

	// Send the next step
	err = svc.executionGateway.SendStepToExecute(result.SagaName, sagaExecution, *nextStep, false)
	if err != nil {
		return err
	}

	return nil
}

func (svc service) onFailureResult(ctx context.Context, result StepResultVO) error {
	// Get Saga Execution
	sagaExecution, err := svc.repository.GetSagaExecution(ctx, result.ExecutionID)
	if err != nil {
		return err
	}

	// Get Step Execution
	stepsExecution, err := svc.repository.GetSagaStepsExecutionByExecutionID(ctx, result.ExecutionID)
	if err != nil {
		return err
	}

	// Mark the received step result as finished
	err = svc.repository.SetSagaStepExecutionStatus(
		ctx, entities.StepExecutionError, result.StepIndex, result.ExecutionID,
	)
	if err != nil {
		return err
	}

	// Get the next step and mark it as started
	nextStep := findPreviousStep(result.StepIndex, stepsExecution)
	if nextStep == nil {
		// Make something better later
		log.Println("Saga has nothing to compensate")
		return nil
	}
	err = svc.repository.SetSagaStepExecutionStatus(
		ctx, entities.StepExecutionInCompensation, nextStep.Index, result.ExecutionID,
	)
	if err != nil {
		return err
	}

	// Send the next step
	err = svc.executionGateway.SendStepToExecute(result.SagaName, sagaExecution, *nextStep, true)
	if err != nil {
		return err
	}

	return nil
}

func (svc service) onCompensation(ctx context.Context, result StepResultVO) error {
	// Get Saga Execution
	sagaExecution, err := svc.repository.GetSagaExecution(ctx, result.ExecutionID)
	if err != nil {
		return err
	}

	// Get Step Execution
	stepsExecution, err := svc.repository.GetSagaStepsExecutionByExecutionID(ctx, result.ExecutionID)
	if err != nil {
		return err
	}

	// Mark the received step result as finished
	err = svc.repository.SetSagaStepExecutionStatus(
		ctx, entities.StepExecutionCompensated, result.StepIndex, result.ExecutionID,
	)
	if err != nil {
		return err
	}

	// Get the next step and mark it as started
	nextStep := findPreviousStep(result.StepIndex, stepsExecution)
	if nextStep == nil {
		// Make something better later
		log.Println("Saga finished compensation")
		return nil
	}
	err = svc.repository.SetSagaStepExecutionStatus(
		ctx, entities.StepExecutionInCompensation, nextStep.Index, result.ExecutionID,
	)
	if err != nil {
		return err
	}

	// Send the next step
	err = svc.executionGateway.SendStepToExecute(result.SagaName, sagaExecution, *nextStep, true)
	if err != nil {
		return err
	}

	return nil
}

func findNextStep(currentIndexStep int, steps []entities.StepExecution) *entities.StepExecution {
	if currentIndexStep < len(steps) {
		return &steps[currentIndexStep]
	}

	return nil
}

func findPreviousStep(currentIndexStep int, steps []entities.StepExecution) *entities.StepExecution {
	stepIndex := currentIndexStep - 2
	if stepIndex >= 0 {
		return &steps[stepIndex]
	}

	return nil
}
