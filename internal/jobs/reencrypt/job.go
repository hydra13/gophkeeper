package reencrypt

import "context"

// Job фоновая ротация и перешифрование данных.
type Job struct{}

// New создаёт job.
func New() *Job {
	return &Job{}
}

// Start запускает job.
func (j *Job) Start(context.Context) error {
	return nil
}

// Stop останавливает job.
func (j *Job) Stop(context.Context) error {
	return nil
}
