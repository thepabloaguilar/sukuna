CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE sagas (
    saga_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    formatted_name TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE saga_steps (
    step_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    saga_id uuid,
    index INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY(saga_id) REFERENCES sagas(saga_id)
);

CREATE TABLE saga_executions (
    saga_execution_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    saga_id uuid,
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(saga_id) REFERENCES sagas(saga_id)
);

CREATE TABLE step_executions (
    step_execution_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    saga_execution_id uuid,
    index INTEGER NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    FOREIGN KEY(saga_execution_id) REFERENCES saga_executions(saga_execution_id)
);
