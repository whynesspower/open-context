package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/opencontext/backend/internal/graphiti"
	"github.com/opencontext/backend/internal/store"
)

func (a *API) postUsers(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID                 string         `json:"user_id"`
		Email                  string         `json:"email"`
		FirstName              string         `json:"first_name"`
		LastName               string         `json:"last_name"`
		Metadata               map[string]any `json:"metadata"`
		DisableDefaultOntology *bool          `json:"disable_default_ontology"`
	}
	if err := a.readJSON(r, &body); err != nil || body.UserID == "" {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	_ = body.DisableDefaultOntology
	u := &store.User{
		UserID:      body.UserID,
		Email:       body.Email,
		FirstName:   body.FirstName,
		LastName:    body.LastName,
		ProjectUUID: a.DB.Project,
		Metadata:    body.Metadata,
		CreatedAt:   a.now(),
		UpdatedAt:   a.now(),
	}
	if _, err := a.DB.NewInsert().Model(u).Exec(r.Context()); err != nil {
		a.err(w, http.StatusBadRequest, "user exists or db error")
		return
	}
	_ = a.G.AddEntityNode(r.Context(), graphiti.AddEntityNodeRequest{
		UUID:    uuid.NewString(),
		GroupID: body.UserID,
		Name:    body.UserID,
		Summary: "user",
	})
	a.json(w, http.StatusOK, userToJSON(u))
}

func (a *API) getUsersOrdered(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page_number"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if size <= 0 {
		size = 50
	}
	var users []store.User
	q := a.DB.NewSelect().Model(&users).Where("project_uuid = ?", a.DB.Project).Order("id ASC").Limit(size).Offset((page - 1) * size)
	count, _ := a.DB.NewSelect().Model((*store.User)(nil)).Where("project_uuid = ?", a.DB.Project).Count(r.Context())
	if err := q.Scan(r.Context()); err != nil {
		a.err(w, http.StatusInternalServerError, "db error")
		return
	}
	out := make([]any, 0, len(users))
	for i := range users {
		out = append(out, userToJSON(&users[i]))
	}
	a.json(w, http.StatusOK, map[string]any{"users": out, "total_count": count, "row_count": len(out)})
}

func (a *API) getUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "userId")
	var u store.User
	err := a.DB.NewSelect().Model(&u).Where("user_id = ? AND project_uuid = ?", id, a.DB.Project).Scan(r.Context())
	if err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	a.json(w, http.StatusOK, userToJSON(&u))
}

func (a *API) patchUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "userId")
	var body map[string]any
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	var u store.User
	if err := a.DB.NewSelect().Model(&u).Where("user_id = ? AND project_uuid = ?", id, a.DB.Project).Scan(r.Context()); err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	if v, ok := body["email"].(string); ok {
		u.Email = v
	}
	if v, ok := body["first_name"].(string); ok {
		u.FirstName = v
	}
	if v, ok := body["last_name"].(string); ok {
		u.LastName = v
	}
	if v, ok := body["metadata"].(map[string]any); ok {
		u.Metadata = v
	}
	u.UpdatedAt = a.now()
	if _, err := a.DB.NewUpdate().Model(&u).WherePK().Exec(r.Context()); err != nil {
		a.err(w, http.StatusInternalServerError, "update failed")
		return
	}
	a.json(w, http.StatusOK, userToJSON(&u))
}

func (a *API) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "userId")
	var u store.User
	if err := a.DB.NewSelect().Model(&u).Where("user_id = ? AND project_uuid = ?", id, a.DB.Project).Scan(r.Context()); err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	_ = a.G.DeleteGroup(r.Context(), id)
	if _, err := a.DB.NewDelete().Model(&u).WherePK().Exec(r.Context()); err != nil {
		a.err(w, http.StatusInternalServerError, "delete failed")
		return
	}
	a.json(w, http.StatusOK, map[string]any{"message": "ok"})
}

func (a *API) getUserThreads(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "userId")
	var sessions []store.Session
	err := a.DB.NewSelect().Model(&sessions).Where("user_id = ? AND project_uuid = ?", id, a.DB.Project).Order("created_at DESC").Scan(r.Context())
	if err != nil {
		a.err(w, http.StatusInternalServerError, "db error")
		return
	}
	out := make([]any, 0, len(sessions))
	for i := range sessions {
		out = append(out, threadToJSON(&sessions[i], a.DB.Project))
	}
	a.json(w, http.StatusOK, out)
}

func (a *API) getUserNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "userId")
	nodes, err := a.G.ListNodes(r.Context(), id, 100)
	if err != nil {
		a.err(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(nodes) == 0 {
		a.json(w, http.StatusOK, map[string]any{"node": nil})
		return
	}
	n := nodes[0]
	a.json(w, http.StatusOK, map[string]any{"node": map[string]any{
		"uuid": n.UUID, "name": n.Name, "summary": n.Summary, "labels": n.Labels, "created_at": n.CreatedAt,
	}})
}

func (a *API) warmUser(w http.ResponseWriter, r *http.Request) {
	a.json(w, http.StatusOK, map[string]any{"message": "warmed", "success": true})
}

func (a *API) listThreads(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page_number"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if size <= 0 {
		size = 50
	}
	var sessions []store.Session
	err := a.DB.NewSelect().Model(&sessions).Where("project_uuid = ?", a.DB.Project).Order("created_at DESC").Limit(size).Offset((page - 1) * size).Scan(r.Context())
	if err != nil {
		a.err(w, http.StatusInternalServerError, "db error")
		return
	}
	total, _ := a.DB.NewSelect().Model((*store.Session)(nil)).Where("project_uuid = ?", a.DB.Project).Count(r.Context())
	out := make([]any, 0, len(sessions))
	for i := range sessions {
		out = append(out, threadToJSON(&sessions[i], a.DB.Project))
	}
	a.json(w, http.StatusOK, map[string]any{"threads": out, "total_count": total, "response_count": len(out)})
}

func (a *API) createThread(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ThreadID string `json:"thread_id"`
		UserID   string `json:"user_id"`
	}
	if err := a.readJSON(r, &body); err != nil || body.ThreadID == "" || body.UserID == "" {
		a.err(w, http.StatusBadRequest, "thread_id and user_id required")
		return
	}
	var u store.User
	if err := a.DB.NewSelect().Model(&u).Where("user_id = ? AND project_uuid = ?", body.UserID, a.DB.Project).Scan(r.Context()); err != nil {
		a.err(w, http.StatusBadRequest, "user not found")
		return
	}
	s := &store.Session{
		SessionID:   body.ThreadID,
		UserID:      &body.UserID,
		ProjectUUID: a.DB.Project,
		CreatedAt:   a.now(),
		UpdatedAt:   a.now(),
	}
	if _, err := a.DB.NewInsert().Model(s).Exec(r.Context()); err != nil {
		a.err(w, http.StatusBadRequest, "thread exists or db error")
		return
	}
	_ = a.G.AddEntityNode(r.Context(), graphiti.AddEntityNodeRequest{
		UUID:    uuid.NewString(),
		GroupID: body.ThreadID,
		Name:    body.ThreadID,
		Summary: "thread",
	})
	a.json(w, http.StatusOK, threadToJSON(s, a.DB.Project))
}

func (a *API) deleteThread(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "threadId")
	var s store.Session
	if err := a.DB.NewSelect().Model(&s).Where("session_id = ? AND project_uuid = ?", id, a.DB.Project).Scan(r.Context()); err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	_ = a.G.DeleteGroup(r.Context(), id)
	if _, err := a.DB.NewDelete().Model(&s).WherePK().Exec(r.Context()); err != nil {
		a.err(w, http.StatusInternalServerError, "delete failed")
		return
	}
	a.json(w, http.StatusOK, map[string]any{"message": "ok"})
}

func (a *API) getThreadMessages(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "threadId")
	var msgs []store.Message
	err := a.DB.NewSelect().Model(&msgs).Where("session_id = ? AND project_uuid = ?", id, a.DB.Project).Order("id ASC").Scan(r.Context())
	if err != nil {
		a.err(w, http.StatusInternalServerError, "db error")
		return
	}
	out := make([]any, 0, len(msgs))
	for i := range msgs {
		out = append(out, messageToJSON(&msgs[i]))
	}
	a.json(w, http.StatusOK, map[string]any{
		"messages":    out,
		"row_count":   len(out),
		"total_count": len(out),
	})
}

func (a *API) addThreadMessages(w http.ResponseWriter, r *http.Request) {
	a.addThreadMessagesInner(w, r, false)
}

func (a *API) addThreadMessagesBatch(w http.ResponseWriter, r *http.Request) {
	a.addThreadMessagesInner(w, r, true)
}

func (a *API) addThreadMessagesInner(w http.ResponseWriter, r *http.Request, _ bool) {
	tid := chi.URLParam(r, "threadId")
	var body struct {
		Messages    []struct {
			Content  string         `json:"content"`
			Role     string         `json:"role"`
			Metadata map[string]any `json:"metadata"`
			Name     string         `json:"name"`
			UUID     string         `json:"uuid"`
		} `json:"messages"`
		ReturnContext *bool `json:"return_context"`
	}
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	var s store.Session
	if err := a.DB.NewSelect().Model(&s).Where("session_id = ? AND project_uuid = ?", tid, a.DB.Project).Scan(r.Context()); err != nil {
		a.err(w, http.StatusNotFound, "thread not found")
		return
	}
	var uuids []string
	gmsgs := make([]graphiti.GMessage, 0, len(body.Messages))
	for _, m := range body.Messages {
		mid := m.UUID
		if mid == "" {
			mid = uuid.NewString()
		}
		muid, err := uuid.Parse(mid)
		if err != nil {
			a.err(w, http.StatusBadRequest, "invalid message uuid")
			return
		}
		msg := &store.Message{
			UUID:        muid,
			SessionID:   tid,
			ProjectUUID: a.DB.Project,
			Role:        m.Role,
			RoleType:    normalizeRoleType(m.Role),
			Content:     m.Content,
			TokenCount:  len(m.Content) / 4,
			Metadata:    m.Metadata,
			Name:        m.Name,
			CreatedAt:   a.now(),
		}
		if _, err := a.DB.NewInsert().Model(msg).Exec(r.Context()); err != nil {
			a.err(w, http.StatusBadRequest, "message insert failed")
			return
		}
		uuids = append(uuids, mid)
		gmsgs = append(gmsgs, graphitiMessageFromStore(msg))
	}
	_ = a.G.AddMessages(r.Context(), tid, gmsgs)
	resp := map[string]any{"message_uuids": uuids}
	if body.ReturnContext != nil && *body.ReturnContext {
		mem, err := a.G.GetMemory(r.Context(), graphiti.GetMemoryRequest{
			GroupID:  tid,
			MaxFacts: 10,
			Messages: gmsgs,
		})
		if err == nil && mem != nil {
			ctx := ""
			for _, f := range mem.Facts {
				ctx += f.Fact + "\n"
			}
			resp["context"] = ctx
		}
	}
	a.json(w, http.StatusOK, resp)
}

func (a *API) getThreadContext(w http.ResponseWriter, r *http.Request) {
	tid := chi.URLParam(r, "threadId")
	var msgs []store.Message
	_ = a.DB.NewSelect().Model(&msgs).Where("session_id = ? AND project_uuid = ?", tid, a.DB.Project).Order("id DESC").Limit(8).Scan(r.Context())
	gmsgs := make([]graphiti.GMessage, 0, len(msgs))
	for i := len(msgs) - 1; i >= 0; i-- {
		gmsgs = append(gmsgs, graphitiMessageFromStore(&msgs[i]))
	}
	mem, err := a.G.GetMemory(r.Context(), graphiti.GetMemoryRequest{
		GroupID:  tid,
		MaxFacts: 20,
		Messages: gmsgs,
	})
	if err != nil {
		a.err(w, http.StatusInternalServerError, err.Error())
		return
	}
	ctx := ""
	for _, f := range mem.Facts {
		ctx += f.Fact + "\n"
	}
	a.json(w, http.StatusOK, map[string]any{"context": ctx, "facts": mem.Facts})
}

func (a *API) patchMessage(w http.ResponseWriter, r *http.Request) {
	mid := chi.URLParam(r, "messageUuid")
	var body struct {
		Metadata map[string]any `json:"metadata"`
	}
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	muid, err := uuid.Parse(mid)
	if err != nil {
		a.err(w, http.StatusBadRequest, "invalid uuid")
		return
	}
	var m store.Message
	err = a.DB.NewSelect().Model(&m).Where("uuid = ? AND project_uuid = ?", muid, a.DB.Project).Scan(r.Context())
	if err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	if m.Metadata == nil {
		m.Metadata = map[string]any{}
	}
	for k, v := range body.Metadata {
		m.Metadata[k] = v
	}
	m.UpdatedAt = a.now()
	if _, err := a.DB.NewUpdate().Model(&m).WherePK().Exec(r.Context()); err != nil {
		a.err(w, http.StatusInternalServerError, "update failed")
		return
	}
	a.json(w, http.StatusOK, messageToJSON(&m))
}

func (a *API) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now().UTC()
}

func userToJSON(u *store.User) map[string]any {
	return map[string]any{
		"user_id":      u.UserID,
		"email":        u.Email,
		"first_name":   u.FirstName,
		"last_name":    u.LastName,
		"metadata":     u.Metadata,
		"project_uuid": u.ProjectUUID.String(),
		"created_at":   ts(u.CreatedAt),
		"updated_at":   ts(u.UpdatedAt),
		"uuid":         u.UUID.String(),
	}
}

func threadToJSON(s *store.Session, proj uuid.UUID) map[string]any {
	out := map[string]any{
		"thread_id":    s.SessionID,
		"project_uuid": proj.String(),
		"created_at":   ts(s.CreatedAt),
		"uuid":         s.UUID.String(),
	}
	if s.UserID != nil {
		out["user_id"] = *s.UserID
	}
	return out
}

func messageToJSON(m *store.Message) map[string]any {
	return map[string]any{
		"uuid":       m.UUID.String(),
		"content":    m.Content,
		"role":       m.Role,
		"role_type":  m.RoleType,
		"metadata":   m.Metadata,
		"name":       m.Name,
		"created_at": ts(m.CreatedAt),
		"processed":  m.Processed,
	}
}

func normalizeRoleType(role string) string {
	switch role {
	case "system", "assistant", "user", "norole", "function", "tool":
		return role
	default:
		return "user"
	}
}

func graphitiMessageFromStore(m *store.Message) graphiti.GMessage {
	rt := m.RoleType
	if rt == "" {
		rt = "user"
	}
	gt := graphitiRoleType(rt)
	role := m.Role
	if role == "" {
		role = rt
	}
	return graphiti.GMessage{
		Content:   m.Content,
		UUID:      m.UUID.String(),
		Name:      m.Name,
		RoleType:  gt,
		Role:      role,
		Timestamp: ts(m.CreatedAt),
	}
}

func graphitiRoleType(rt string) string {
	switch rt {
	case "system", "assistant", "user":
		return rt
	default:
		return "user"
	}
}
