package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Bhanubpsn/taskflow-bhanupratap-singh-negi/models"
)

func GetTasks() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := currentUserID(c)
		if !ok {
			return
		}
		projectID, ok := parseUUIDParam(c, "id")
		if !ok {
			return
		}

		query := `
			SELECT id, title, description, status, priority,
			       project_id, assignee_id, due_date, created_at, updated_at, created_by
			FROM tasks
			WHERE project_id = $1`
		args := []interface{}{projectID}
		i := 2

		if status := c.Query("status"); status != "" {
			query += fmt.Sprintf(" AND status = $%d", i)
			args = append(args, status)
			i++
		}
		if assigneeStr := c.Query("assignee"); assigneeStr != "" {
			assigneeID, err := uuid.Parse(assigneeStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignee id"})
				return
			}
			query += fmt.Sprintf(" AND assignee_id = $%d", i)
			args = append(args, assigneeID)
		}
		query += " ORDER BY created_at ASC"

		rows, err := DB.Query(c.Request.Context(), query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch tasks"})
			return
		}
		defer rows.Close()

		tasks := []gin.H{}
		for rows.Next() {
			t, err := scanTask(rows)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read tasks"})
				return
			}
			tasks = append(tasks, taskJSON(t))
		}

		c.JSON(http.StatusOK, tasks)
	}
}

func CreateTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := currentUserID(c)
		if !ok {
			return
		}
		projectID, ok := parseUUIDParam(c, "id")
		if !ok {
			return
		}

		// Only the project owner can create tasks
		var ownerID uuid.UUID
		err := DB.QueryRow(c.Request.Context(),
			`SELECT owner_id FROM projects WHERE id = $1`, projectID,
		).Scan(&ownerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		if ownerID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the project owner can create tasks"})
			return
		}

		var input struct {
			Title       string               `json:"title"        binding:"required"`
			Description *string              `json:"description"`
			Status      *models.TaskStatus   `json:"status"`
			Priority    *models.TaskPriority `json:"priority"`
			AssigneeID  *uuid.UUID           `json:"assignee_id"`
			DueDate     *time.Time           `json:"due_date"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		status := models.StatusTodo
		if input.Status != nil {
			status = *input.Status
		}
		priority := models.PriorityMedium
		if input.Priority != nil {
			priority = *input.Priority
		}

		var t models.Task
		err = DB.QueryRow(c.Request.Context(), `
			INSERT INTO tasks
			    (title, description, status, priority, project_id, assignee_id, due_date, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, title, description, status, priority,
			          project_id, assignee_id, due_date, created_at, updated_at, created_by`,
			input.Title, input.Description, status, priority,
			projectID, input.AssigneeID, input.DueDate, userID,
		).Scan(
			&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
			&t.ProjectID, &t.AssigneeID, &t.DueDate,
			&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create task"})
			return
		}

		c.JSON(http.StatusCreated, taskJSON(t))
	}
}

func UpdateTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := currentUserID(c)
		if !ok {
			return
		}
		taskID, ok := parseUUIDParam(c, "id")
		if !ok {
			return
		}

		var input struct {
			Title       *string              `json:"title"`
			Description *string              `json:"description"`
			Status      *models.TaskStatus   `json:"status"`
			Priority    *models.TaskPriority `json:"priority"`
			AssigneeID  *uuid.UUID           `json:"assignee_id"`
			DueDate     *time.Time           `json:"due_date"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// updated_at is always set; remaining clauses are conditional
		setClauses := []string{"updated_at = NOW()"}
		args := []interface{}{}
		i := 1

		if input.Title != nil {
			setClauses = append(setClauses, fmt.Sprintf("title = $%d", i))
			args = append(args, *input.Title)
			i++
		}
		if input.Description != nil {
			setClauses = append(setClauses, fmt.Sprintf("description = $%d", i))
			args = append(args, *input.Description)
			i++
		}
		if input.Status != nil {
			setClauses = append(setClauses, fmt.Sprintf("status = $%d", i))
			args = append(args, *input.Status)
			i++
		}
		if input.Priority != nil {
			setClauses = append(setClauses, fmt.Sprintf("priority = $%d", i))
			args = append(args, *input.Priority)
			i++
		}
		if input.AssigneeID != nil {
			setClauses = append(setClauses, fmt.Sprintf("assignee_id = $%d", i))
			args = append(args, *input.AssigneeID)
			i++
		}
		if input.DueDate != nil {
			setClauses = append(setClauses, fmt.Sprintf("due_date = $%d", i))
			args = append(args, *input.DueDate)
			i++
		}
		if len(setClauses) == 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}

		args = append(args, taskID)
		query := fmt.Sprintf(`
			UPDATE tasks SET %s WHERE id = $%d
			RETURNING id, title, description, status, priority,
			          project_id, assignee_id, due_date, created_at, updated_at, created_by`,
			strings.Join(setClauses, ", "), i,
		)

		var t models.Task
		err := DB.QueryRow(c.Request.Context(), query, args...).Scan(
			&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
			&t.ProjectID, &t.AssigneeID, &t.DueDate,
			&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy,
		)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		c.JSON(http.StatusOK, taskJSON(t))
	}
}

func DeleteTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := currentUserID(c)
		if !ok {
			return
		}
		taskID, ok := parseUUIDParam(c, "id")
		if !ok {
			return
		}

		var createdBy *uuid.UUID
		var ownerID uuid.UUID
		err := DB.QueryRow(c.Request.Context(), `
			SELECT t.created_by, p.owner_id
			FROM tasks t
			JOIN projects p ON p.id = t.project_id
			WHERE t.id = $1`,
			taskID,
		).Scan(&createdBy, &ownerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		isOwner := ownerID == userID
		isCreator := createdBy != nil && *createdBy == userID
		if !isOwner && !isCreator {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the project owner or task creator can delete this task"})
			return
		}

		if _, err = DB.Exec(c.Request.Context(),
			`DELETE FROM tasks WHERE id = $1`, taskID,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete task"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "task deleted"})
	}
}
