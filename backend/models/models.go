package models

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string
type TaskPriority string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

const (
	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
)

type User struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	Email     string    `db:"email"`
	Password  string    `db:"password"`
	CreatedAt time.Time `db:"created_at"`
}

type Project struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	Description *string   `db:"description"`
	OwnerID     uuid.UUID `db:"owner_id"`
	CreatedAt   time.Time `db:"created_at"`
}

type Task struct {
	ID          uuid.UUID    `db:"id"`
	Title       string       `db:"title"`
	Description *string      `db:"description"`
	Status      TaskStatus   `db:"status"`
	Priority    TaskPriority `db:"priority"`
	ProjectID   uuid.UUID    `db:"project_id"`
	AssigneeID  *uuid.UUID   `db:"assignee_id"`
	DueDate     *time.Time   `db:"due_date"`
	CreatedAt   time.Time    `db:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at"`
	CreatedBy   *uuid.UUID   `db:"created_by"`
}
