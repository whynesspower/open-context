package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/opencontext/backend/internal/store"
)

func (a *API) projectInfo(w http.ResponseWriter, r *http.Request) {
	a.json(w, http.StatusOK, map[string]any{
		"project_name": a.Cfg.OpenContextName,
		"version":      a.Cfg.OpenContextVersion,
		"project_uuid": a.DB.Project.String(),
	})
}

func (a *API) getTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	var t store.TaskRecord
	err := a.DB.NewSelect().Model(&t).Where("task_id = ? AND project_uuid = ?", id, a.DB.Project).Scan(r.Context())
	if err != nil {
		a.json(w, http.StatusOK, map[string]any{"task_id": id, "status": "completed", "progress": 1.0})
		return
	}
	a.json(w, http.StatusOK, map[string]any{"task_id": t.TaskID, "status": t.Status, "progress": t.Progress, "error": t.Error})
}

func (a *API) listContextTemplates(w http.ResponseWriter, r *http.Request) {
	var rows []store.ContextTemplate
	_ = a.DB.NewSelect().Model(&rows).Where("project_uuid = ?", a.DB.Project).Scan(r.Context())
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		out = append(out, map[string]any{
			"template_id": rows[i].ID, "name": rows[i].Name, "content": rows[i].Content,
			"created_at": ts(rows[i].CreatedAt), "updated_at": ts(rows[i].UpdatedAt),
		})
	}
	a.json(w, http.StatusOK, map[string]any{"templates": out})
}

func (a *API) createContextTemplate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TemplateID string `json:"template_id"`
		Name       string `json:"name"`
		Content    string `json:"content"`
	}
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.TemplateID == "" {
		body.TemplateID = uuid.NewString()
	}
	row := &store.ContextTemplate{
		ID:          body.TemplateID,
		Name:        body.Name,
		Content:     body.Content,
		ProjectUUID: a.DB.Project,
		CreatedAt:   a.now(),
		UpdatedAt:   a.now(),
	}
	if _, err := a.DB.NewInsert().Model(row).Exec(r.Context()); err != nil {
		a.err(w, http.StatusBadRequest, "exists")
		return
	}
	a.json(w, http.StatusOK, map[string]any{
		"template_id": row.ID, "name": row.Name, "content": row.Content,
		"created_at": ts(row.CreatedAt), "updated_at": ts(row.UpdatedAt),
	})
}

func (a *API) getContextTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "templateId")
	var row store.ContextTemplate
	if err := a.DB.NewSelect().Model(&row).Where("id = ? AND project_uuid = ?", id, a.DB.Project).Scan(r.Context()); err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	a.json(w, http.StatusOK, map[string]any{
		"template_id": row.ID, "name": row.Name, "content": row.Content,
		"created_at": ts(row.CreatedAt), "updated_at": ts(row.UpdatedAt),
	})
}

func (a *API) updateContextTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "templateId")
	var body map[string]any
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	var row store.ContextTemplate
	if err := a.DB.NewSelect().Model(&row).Where("id = ? AND project_uuid = ?", id, a.DB.Project).Scan(r.Context()); err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	if v, ok := body["name"].(string); ok {
		row.Name = v
	}
	if v, ok := body["content"].(string); ok {
		row.Content = v
	}
	row.UpdatedAt = a.now()
	_, _ = a.DB.NewUpdate().Model(&row).WherePK().Exec(r.Context())
	a.json(w, http.StatusOK, map[string]any{
		"template_id": row.ID, "name": row.Name, "content": row.Content,
		"created_at": ts(row.CreatedAt), "updated_at": ts(row.UpdatedAt),
	})
}

func (a *API) deleteContextTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "templateId")
	if _, err := a.DB.NewDelete().Model((*store.ContextTemplate)(nil)).Where("id = ? AND project_uuid = ?", id, a.DB.Project).Exec(r.Context()); err != nil {
		a.err(w, http.StatusInternalServerError, "delete failed")
		return
	}
	a.json(w, http.StatusOK, map[string]any{"message": "ok"})
}

func (a *API) listCustomInstructions(w http.ResponseWriter, r *http.Request) {
	var rows []store.CustomInstructionRow
	_ = a.DB.NewSelect().Model(&rows).Where("project_uuid = ?", a.DB.Project).Scan(r.Context())
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		out = append(out, map[string]any{"name": rows[i].Name, "text": rows[i].Text})
	}
	a.json(w, http.StatusOK, map[string]any{"instructions": out})
}

func (a *API) addCustomInstructions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Instructions []struct {
			Name string `json:"name"`
			Text string `json:"text"`
		} `json:"instructions"`
	}
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	for _, ins := range body.Instructions {
		row := &store.CustomInstructionRow{
			Name: ins.Name, Text: ins.Text, Scope: "project", ProjectUUID: a.DB.Project, CreatedAt: a.now(),
		}
		_, _ = a.DB.NewInsert().Model(row).Exec(r.Context())
	}
	a.json(w, http.StatusOK, map[string]any{"message": "ok"})
}

func (a *API) deleteCustomInstructions(w http.ResponseWriter, r *http.Request) {
	_, _ = a.DB.NewDelete().Model((*store.CustomInstructionRow)(nil)).Where("project_uuid = ?", a.DB.Project).Exec(r.Context())
	a.json(w, http.StatusOK, map[string]any{"message": "ok"})
}

func (a *API) listUserSummaryInstructions(w http.ResponseWriter, r *http.Request) {
	var rows []store.UserSummaryInstructionRow
	_ = a.DB.NewSelect().Model(&rows).Where("project_uuid = ?", a.DB.Project).Scan(r.Context())
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		out = append(out, map[string]any{"name": rows[i].Name, "text": rows[i].Text})
	}
	a.json(w, http.StatusOK, map[string]any{"instructions": out})
}

func (a *API) addUserSummaryInstructions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Instructions []struct {
			Name string `json:"name"`
			Text string `json:"text"`
		} `json:"instructions"`
	}
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	for _, ins := range body.Instructions {
		row := &store.UserSummaryInstructionRow{Name: ins.Name, Text: ins.Text, ProjectUUID: a.DB.Project, CreatedAt: a.now()}
		_, _ = a.DB.NewInsert().Model(row).Exec(r.Context())
	}
	a.json(w, http.StatusOK, map[string]any{"message": "ok"})
}

func (a *API) deleteUserSummaryInstructions(w http.ResponseWriter, r *http.Request) {
	_, _ = a.DB.NewDelete().Model((*store.UserSummaryInstructionRow)(nil)).Where("project_uuid = ?", a.DB.Project).Exec(r.Context())
	a.json(w, http.StatusOK, map[string]any{"message": "ok"})
}

func (a *API) listEntityTypes(w http.ResponseWriter, r *http.Request) {
	var row store.EntityTypesRow
	err := a.DB.NewSelect().Model(&row).Where("project_uuid = ?", a.DB.Project).Scan(r.Context())
	if err != nil {
		a.json(w, http.StatusOK, map[string]any{"entity_types": []any{}, "edge_types": []any{}})
		return
	}
	if row.Payload != nil {
		a.json(w, http.StatusOK, row.Payload)
		return
	}
	a.json(w, http.StatusOK, map[string]any{"entity_types": []any{}, "edge_types": []any{}})
}

func (a *API) putEntityTypes(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	_, _ = a.DB.NewDelete().Model((*store.EntityTypesRow)(nil)).Where("project_uuid = ?", a.DB.Project).Exec(r.Context())
	row := &store.EntityTypesRow{ProjectUUID: a.DB.Project, Payload: body, UpdatedAt: a.now()}
	if _, err := a.DB.NewInsert().Model(row).Exec(r.Context()); err != nil {
		a.err(w, http.StatusInternalServerError, "save failed")
		return
	}
	a.json(w, http.StatusOK, map[string]any{"message": "ok"})
}
