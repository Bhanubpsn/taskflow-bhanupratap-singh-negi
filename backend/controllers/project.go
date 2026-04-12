package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Bhanubpsn/taskflow-bhanupratap-singh-negi/models"
)

func GetProjects() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := currentUserID(c)
		if !ok {
			return
		}

		rows, err := DB.Query(c.Request.Context(), `
			SELECT p.id, p.name, p.description, p.owner_id, p.created_at
			FROM projects p
			WHERE p.owner_id = $1
			   OR EXISTS (
			       SELECT 1 FROM tasks t
			       WHERE t.project_id = p.id AND t.assignee_id = $1
			   )
			ORDER BY p.created_at DESC`,
			userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch projects"})
			return
		}
		defer rows.Close()

		projects := []gin.H{}
		for rows.Next() {
			var p models.Project
			if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read projects"})
				return
			}
			projects = append(projects, projectJSON(p))
		}

		c.JSON(http.StatusOK, projects)
	}
}

func CreateProject() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := currentUserID(c)
		if !ok {
			return
		}

		var input struct {
			Name        string  `json:"name"        binding:"required"`
			Description *string `json:"description"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var p models.Project
		err := DB.QueryRow(c.Request.Context(), `
			INSERT INTO projects (name, description, owner_id)
			VALUES ($1, $2, $3)
			RETURNING id, name, description, owner_id, created_at`,
			input.Name, input.Description, userID,
		).Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create project"})
			return
		}

		c.JSON(http.StatusCreated, projectJSON(p))
	}
}

func GetProjectByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := currentUserID(c)
		if !ok {
			return
		}
		projectID, ok := parseUUIDParam(c, "id")
		if !ok {
			return
		}

		// Fetch project — only visible to owner or anyone assigned to a task in it
		var p models.Project
		err := DB.QueryRow(c.Request.Context(), `
			SELECT id, name, description, owner_id, created_at
			FROM projects
			WHERE id = $1
			  AND (owner_id = $2 OR EXISTS (
			      SELECT 1 FROM tasks t WHERE t.project_id = $1 AND t.assignee_id = $2
			  ))`,
			projectID, userID,
		).Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}

		rows, err := DB.Query(c.Request.Context(), `
			SELECT id, title, description, status, priority,
			       project_id, assignee_id, due_date, created_at, updated_at, created_by
			FROM tasks
			WHERE project_id = $1
			ORDER BY created_at ASC`,
			projectID,
		)
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

		result := projectJSON(p)
		result["tasks"] = tasks
		c.JSON(http.StatusOK, result)
	}
}

func UpdateProject() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := currentUserID(c)
		if !ok {
			return
		}
		projectID, ok := parseUUIDParam(c, "id")
		if !ok {
			return
		}

		var ownerID uuid.UUID
		err := DB.QueryRow(c.Request.Context(),
			`SELECT owner_id FROM projects WHERE id = $1`, projectID,
		).Scan(&ownerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		if ownerID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the project owner can update this project"})
			return
		}

		var input struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		setClauses := []string{}
		args := []interface{}{}
		i := 1

		if input.Name != nil {
			setClauses = append(setClauses, fmt.Sprintf("name = $%d", i))
			args = append(args, *input.Name)
			i++
		}
		if input.Description != nil {
			setClauses = append(setClauses, fmt.Sprintf("description = $%d", i))
			args = append(args, *input.Description)
			i++
		}
		if len(setClauses) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}

		args = append(args, projectID)
		query := fmt.Sprintf(
			`UPDATE projects SET %s WHERE id = $%d
			 RETURNING id, name, description, owner_id, created_at`,
			strings.Join(setClauses, ", "), i,
		)

		var p models.Project
		err = DB.QueryRow(c.Request.Context(), query, args...).
			Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update project"})
			return
		}

		c.JSON(http.StatusOK, projectJSON(p))
	}
}

func DeleteProject() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := currentUserID(c)
		if !ok {
			return
		}
		projectID, ok := parseUUIDParam(c, "id")
		if !ok {
			return
		}

		var ownerID uuid.UUID
		err := DB.QueryRow(c.Request.Context(),
			`SELECT owner_id FROM projects WHERE id = $1`, projectID,
		).Scan(&ownerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		if ownerID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the project owner can delete this project"})
			return
		}

		if _, err = DB.Exec(c.Request.Context(),
			`DELETE FROM projects WHERE id = $1`, projectID,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete project"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "project deleted"})
	}
}

func projectJSON(p models.Project) gin.H {
	return gin.H{
		"id":          p.ID,
		"name":        p.Name,
		"description": p.Description,
		"owner_id":    p.OwnerID,
		"created_at":  p.CreatedAt,
	}
}

func scanTask(rows pgx.Rows) (models.Task, error) {
	var t models.Task
	err := rows.Scan(
		&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
		&t.ProjectID, &t.AssigneeID, &t.DueDate,
		&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy,
	)
	return t, err
}

func taskJSON(t models.Task) gin.H {
	return gin.H{
		"id":          t.ID,
		"title":       t.Title,
		"description": t.Description,
		"status":      t.Status,
		"priority":    t.Priority,
		"project_id":  t.ProjectID,
		"assignee_id": t.AssigneeID,
		"due_date":    t.DueDate,
		"created_at":  t.CreatedAt,
		"updated_at":  t.UpdatedAt,
		"created_by":  t.CreatedBy,
	}
}
