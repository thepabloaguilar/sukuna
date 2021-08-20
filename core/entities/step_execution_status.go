package entities

type StepExecutionStatus string

const (
	StepExecutionRegistered     StepExecutionStatus = "registered"
	StepExecutionStarted        StepExecutionStatus = "started"
	StepExecutionFinished       StepExecutionStatus = "finished"
	StepExecutionCompensated    StepExecutionStatus = "compensated"
	StepExecutionInCompensation StepExecutionStatus = "in_compensation"
	StepExecutionError          StepExecutionStatus = "error"
)
