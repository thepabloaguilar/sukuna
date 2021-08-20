package routes

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/thepabloaguilar/sukuna/core/sagas"
)

func SagaRouter(app fiber.Router, service sagas.Service) {
	// Sagas
	app.Get("/sagas/:sagaID", getSaga(service))
	app.Post("/sagas", createSaga(service))

	// Saga executions
	app.Get("/sagas/:sagaID/executions/:executionID", getSagaExecution(service))
	app.Post("/sagas/:sagaID/executions", createSagaExecution(service))
}

type getSagaResponse struct {
	SagaID        uuid.UUID       `json:"saga_id"`
	Name          string          `json:"name"`
	FormattedName string          `json:"formatted_name"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"created_at"`
}

func getSaga(service sagas.Service) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		stringSagaID := ctx.Params("sagaID")

		sagaID, err := uuid.Parse(stringSagaID)
		if err != nil {
			return ctx.Status(fiber.StatusBadRequest).
				JSON(map[string]string{"error": err.Error()})
		}

		saga, err := service.GetSagaByID(ctx.Context(), sagaID)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).
				JSON(map[string]string{"error": err.Error()})
		}

		response := getSagaResponse{
			SagaID:        saga.SagaID,
			Name:          saga.Name,
			FormattedName: saga.FormattedName,
			Payload:       saga.Payload,
			CreatedAt:     saga.CreatedAt,
		}
		return ctx.JSON(response)
	}
}

type createSagaRequest struct {
	Name    string                   `json:"name" validate:"required"`
	Payload json.RawMessage          `json:"payload" validate:"required"`
	Steps   []createSagaRequestSteps `json:"steps" validate:"required"`
}

type createSagaRequestSteps struct {
	Name string `json:"name" validate:"required"`
}

func (p createSagaRequest) toVO() sagas.CreateSagaVO {
	steps := make([]sagas.CreateSagaVOSteps, 0)
	for _, step := range p.Steps {
		steps = append(steps, sagas.CreateSagaVOSteps{Name: step.Name})
	}

	return sagas.CreateSagaVO{
		Name:    p.Name,
		Payload: p.Payload,
		Steps:   steps,
	}
}

type createSagaResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func createSaga(service sagas.Service) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		payload := new(createSagaRequest)

		if err := ctx.BodyParser(payload); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).
				JSON(map[string]string{"error": err.Error()})
		}

		validationErrors := validateStruct(payload)
		if len(validationErrors) > 0 {
			return ctx.Status(fiber.StatusBadRequest).JSON(validationErrors)
		}

		saga, err := service.CreateSaga(ctx.Context(), payload.toVO())
		if err != nil {
			if errors.Is(err, sagas.ErrInvalidJSONSchema) {
				return ctx.Status(fiber.StatusBadRequest).
					JSON(map[string]string{"error": err.Error()})
			}

			return ctx.Status(fiber.StatusInternalServerError).
				JSON(map[string]string{"error": err.Error()})
		}

		response := createSagaResponse{
			ID:        saga.SagaID,
			Name:      saga.Name,
			CreatedAt: saga.CreatedAt,
		}
		return ctx.Status(fiber.StatusCreated).JSON(response)
	}
}

type getSagaExecutionResponse struct {
	SagaExecutionID uuid.UUID                       `json:"saga_execution_id"`
	SagaID          uuid.UUID                       `json:"saga_id"`
	Payload         json.RawMessage                 `json:"payload"`
	Steps           []getSagaExecutionResponseSteps `json:"steps"`
}

type getSagaExecutionResponseSteps struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func getSagaExecution(service sagas.Service) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		stringExecutionID := ctx.Params("executionID")

		// TODO: extract to a function?
		executionID, err := uuid.Parse(stringExecutionID)
		if err != nil {
			return ctx.Status(fiber.StatusBadRequest).
				JSON(map[string]string{"error": err.Error()})
		}

		execution, err := service.GetSagaExecution(ctx.Context(), executionID)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).
				JSON(map[string]string{"error": err.Error()})
		}

		response := getSagaExecutionResponse{
			SagaExecutionID: execution.SagaExecutionID,
			SagaID:          execution.SagaID,
			Payload:         execution.Payload,
			Steps:           make([]getSagaExecutionResponseSteps, 0, len(execution.Steps)),
		}
		for _, step := range execution.Steps {
			response.Steps = append(response.Steps, getSagaExecutionResponseSteps{
				Name:   step.Name,
				Status: string(step.Status),
			})
		}

		return ctx.JSON(response)
	}
}

type createSagaExecutionRequest struct {
	Payload json.RawMessage `json:"payload"`
}

type createSagaExecutionResponse struct {
	SagaExecutionID uuid.UUID `json:"saga_execution_id"`
	StartedAt       time.Time `json:"started_at"`
}

func createSagaExecution(service sagas.Service) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		stringSagaID := ctx.Params("sagaID")

		// TODO: extract to a function?
		sagaID, err := uuid.Parse(stringSagaID)
		if err != nil {
			return ctx.Status(fiber.StatusBadRequest).
				JSON(map[string]string{"error": err.Error()})
		}

		payload := new(createSagaExecutionRequest)
		if err := ctx.BodyParser(payload); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).
				JSON(map[string]string{"error": err.Error()})
		}

		validationErrors := validateStruct(payload)
		if len(validationErrors) > 0 {
			return ctx.Status(fiber.StatusBadRequest).JSON(validationErrors)
		}

		vo := sagas.CreateSagaExecutionVO{
			SagaID:  sagaID,
			Payload: payload.Payload,
		}
		execution, err := service.CreateSagaExecution(ctx.Context(), vo)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).
				JSON(map[string]string{"error": err.Error()})
		}

		response := createSagaExecutionResponse{
			SagaExecutionID: execution.SagaExecutionID,
			StartedAt:       execution.CreatedAt,
		}
		return ctx.Status(fiber.StatusCreated).JSON(response)
	}
}
