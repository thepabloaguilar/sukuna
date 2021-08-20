-- name: GetSaga :one
SELECT * FROM sagas WHERE saga_id = $1;

-- name: CreateSaga :one
INSERT INTO sagas (name, formatted_name, payload)
VALUES ($1, $2, $3) RETURNING *;

-- name: GetSagaStepsBySagaID :many
SELECT * FROM saga_steps WHERE saga_id = $1;

-- name: CreateSagaSteps :many
INSERT INTO saga_steps(saga_id, index, name)
SELECT
    unnest(@saga_ids::uuid[]) AS saga_id,
    unnest(@indexes::INTEGER[]) as index,
    unnest(@names::TEXT[]) AS name
RETURNING *;

-- name: GetSagaExecution :one
SELECT * FROM saga_executions WHERE saga_execution_id = $1;

-- name: CreateSagaExecution :one
INSERT INTO saga_executions (saga_id, payload)
VALUES ($1, $2) RETURNING *;

-- name: GetSagaStepsExecutionByExecutionID :many
SELECT * FROM step_executions WHERE saga_execution_id = $1 ORDER BY index;

-- name: CreateSagaStepsExecution :many
INSERT INTO step_executions(saga_execution_id, index, name, status)
SELECT
   unnest(@saga_execution_ids::uuid[]) AS saga_execution_id,
   unnest(@indexes::INTEGER[]) as index,
   unnest(@names::TEXT[]) AS name,
   unnest(@statuses::TEXT[]) as status
RETURNING *;

-- name: SetSagaStepExecutionStatus :exec
UPDATE step_executions SET status = $1 WHERE index = $2 AND saga_execution_id = $3;
