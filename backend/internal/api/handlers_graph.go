package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/opencontext/backend/internal/graphiti"
	"github.com/opencontext/backend/internal/store"
)

func (a *API) graphSearch(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	q, _ := body["query"].(string)
	max := 10
	if v, ok := body["limit"].(float64); ok {
		max = int(v)
	}
	var gids []string
	if raw, ok := body["graph_id"].(string); ok && raw != "" {
		gids = append(gids, raw)
	}
	if raw, ok := body["user_id"].(string); ok && raw != "" {
		gids = append(gids, raw)
	}
	if arr, ok := body["group_ids"].([]any); ok {
		for _, x := range arr {
			if s, ok := x.(string); ok {
				gids = append(gids, s)
			}
		}
	}
	res, err := a.G.Search(r.Context(), graphiti.SearchQuery{GroupIDs: gids, Query: q, MaxFacts: max})
	if err != nil {
		a.err(w, http.StatusInternalServerError, err.Error())
		return
	}
	edges := make([]map[string]any, 0, len(res.Facts))
	for _, f := range res.Facts {
		edges = append(edges, factToEdge(f))
	}
	a.json(w, http.StatusOK, map[string]any{"edges": edges, "nodes": []any{}})
}

func factToEdge(f graphiti.FactResult) map[string]any {
	return map[string]any{
		"uuid": f.UUID, "name": f.Name, "fact": f.Fact,
		"created_at": f.CreatedAt, "valid_at": f.ValidAt, "invalid_at": f.InvalidAt, "expired_at": f.ExpiredAt,
		"source_node_uuid": "", "target_node_uuid": "",
	}
}

func (a *API) graphCreate(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	gid, _ := body["graph_id"].(string)
	if gid == "" {
		gid = uuid.NewString()
	}
	rec := &store.GraphRecord{
		GraphID:     gid,
		ProjectUUID: a.DB.Project,
		Metadata:    asMap(body["metadata"]),
	}
	if uid, ok := body["user_id"].(string); ok && uid != "" {
		rec.UserID = &uid
	}
	if _, err := a.DB.NewInsert().Model(rec).Exec(r.Context()); err != nil {
		a.err(w, http.StatusBadRequest, "graph exists")
		return
	}
	_ = a.G.AddEntityNode(r.Context(), graphiti.AddEntityNodeRequest{UUID: uuid.NewString(), GroupID: gid, Name: gid, Summary: "graph"})
	a.json(w, http.StatusOK, graphToJSON(rec))
}

func (a *API) graphListAll(w http.ResponseWriter, r *http.Request) {
	var graphs []store.GraphRecord
	_ = a.DB.NewSelect().Model(&graphs).Where("project_uuid = ?", a.DB.Project).Order("created_at DESC").Limit(500).Scan(r.Context())
	out := make([]any, 0, len(graphs))
	for i := range graphs {
		out = append(out, graphToJSON(&graphs[i]))
	}
	a.json(w, http.StatusOK, map[string]any{"graphs": out})
}

func (a *API) graphGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "graphId")
	var g store.GraphRecord
	if err := a.DB.NewSelect().Model(&g).Where("graph_id = ? AND project_uuid = ?", id, a.DB.Project).Scan(r.Context()); err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	a.json(w, http.StatusOK, graphToJSON(&g))
}

func (a *API) graphPatch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "graphId")
	var body map[string]any
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	var g store.GraphRecord
	if err := a.DB.NewSelect().Model(&g).Where("graph_id = ? AND project_uuid = ?", id, a.DB.Project).Scan(r.Context()); err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	if m := asMap(body["metadata"]); m != nil {
		g.Metadata = m
	}
	g.UpdatedAt = a.now()
	_, _ = a.DB.NewUpdate().Model(&g).WherePK().Exec(r.Context())
	a.json(w, http.StatusOK, graphToJSON(&g))
}

func (a *API) graphDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "graphId")
	var g store.GraphRecord
	if err := a.DB.NewSelect().Model(&g).Where("graph_id = ? AND project_uuid = ?", id, a.DB.Project).Scan(r.Context()); err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	_ = a.G.DeleteGroup(r.Context(), id)
	_, _ = a.DB.NewDelete().Model(&g).WherePK().Exec(r.Context())
	a.json(w, http.StatusOK, map[string]any{"message": "ok"})
}

func (a *API) graphAdd(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	group := firstString(body["graph_id"], body["user_id"])
	if group == "" {
		a.err(w, http.StatusBadRequest, "graph_id or user_id required")
		return
	}
	data, _ := body["data"].(string)
	typ, _ := body["type"].(string)
	gm := graphiti.GMessage{Content: data, RoleType: "user", Role: typ, UUID: uuid.NewString()}
	_ = a.G.AddMessages(r.Context(), group, []graphiti.GMessage{gm})
	a.json(w, http.StatusOK, map[string]any{"uuid": gm.UUID, "content": data, "type": typ})
}

func (a *API) graphAddBatch(w http.ResponseWriter, r *http.Request) {
	a.json(w, http.StatusOK, []any{})
}

func (a *API) graphAddFactTriple(w http.ResponseWriter, r *http.Request) {
	a.json(w, http.StatusOK, map[string]any{"uuid": uuid.NewString(), "message": "queued"})
}

func (a *API) graphClone(w http.ResponseWriter, r *http.Request) {
	a.json(w, http.StatusOK, map[string]any{"message": "cloned"})
}

func (a *API) graphPatterns(w http.ResponseWriter, r *http.Request) {
	a.json(w, http.StatusOK, map[string]any{"patterns": []any{}})
}

func (a *API) postNodesByGraph(w http.ResponseWriter, r *http.Request) {
	gid := chi.URLParam(r, "graphId")
	var body struct {
		Limit      int    `json:"limit"`
		UUIDCursor string `json:"uuid_cursor"`
	}
	_ = a.readJSON(r, &body)
	nodes, err := a.G.ListNodes(r.Context(), gid, body.Limit)
	if err != nil {
		a.err(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.json(w, http.StatusOK, nodesToSDK(nodes))
}

func (a *API) postNodesByUser(w http.ResponseWriter, r *http.Request) {
	uid := chi.URLParam(r, "userId")
	var body struct {
		Limit      int    `json:"limit"`
		UUIDCursor string `json:"uuid_cursor"`
	}
	_ = a.readJSON(r, &body)
	nodes, err := a.G.ListNodes(r.Context(), uid, body.Limit)
	if err != nil {
		a.err(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.json(w, http.StatusOK, nodesToSDK(nodes))
}

func nodesToSDK(nodes []graphiti.GraphitiNode) []map[string]any {
	out := make([]map[string]any, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, map[string]any{
			"uuid": n.UUID, "name": n.Name, "summary": n.Summary, "labels": n.Labels,
			"created_at": n.CreatedAt,
		})
	}
	return out
}

func edgesToSDK(edges []graphiti.GraphitiEdge) []map[string]any {
	out := make([]map[string]any, 0, len(edges))
	for _, e := range edges {
		out = append(out, map[string]any{
			"uuid": e.UUID, "name": e.Name, "fact": e.Fact,
			"source_node_uuid": e.SourceNodeUUID, "target_node_uuid": e.TargetNodeUUID,
			"created_at": e.CreatedAt, "valid_at": e.ValidAt, "invalid_at": e.InvalidAt, "expired_at": e.ExpiredAt,
			"episodes": e.Episodes,
		})
	}
	return out
}

func (a *API) postEdgesByGraph(w http.ResponseWriter, r *http.Request) {
	gid := chi.URLParam(r, "graphId")
	var body struct {
		Limit      int    `json:"limit"`
		UUIDCursor string `json:"uuid_cursor"`
	}
	_ = a.readJSON(r, &body)
	edges, err := a.G.ListEdges(r.Context(), gid, body.Limit)
	if err != nil {
		a.err(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.json(w, http.StatusOK, edgesToSDK(edges))
}

func (a *API) postEdgesByUser(w http.ResponseWriter, r *http.Request) {
	uid := chi.URLParam(r, "userId")
	var body struct {
		Limit      int    `json:"limit"`
		UUIDCursor string `json:"uuid_cursor"`
	}
	_ = a.readJSON(r, &body)
	edges, err := a.G.ListEdges(r.Context(), uid, body.Limit)
	if err != nil {
		a.err(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.json(w, http.StatusOK, edgesToSDK(edges))
}

func (a *API) getNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "nodeUuid")
	a.json(w, http.StatusOK, map[string]any{"uuid": id, "name": "", "summary": "", "labels": []any{}, "created_at": ts(a.now())})
}

func (a *API) patchNode(w http.ResponseWriter, r *http.Request) {
	a.getNode(w, r)
}

func (a *API) deleteNode(w http.ResponseWriter, r *http.Request) {
	a.json(w, http.StatusOK, map[string]any{"message": "ok"})
}

func (a *API) getNodeEdges(w http.ResponseWriter, r *http.Request) {
	a.json(w, http.StatusOK, []any{})
}

func (a *API) getNodeEpisodes(w http.ResponseWriter, r *http.Request) {
	a.json(w, http.StatusOK, map[string]any{"episodes": []any{}})
}

func (a *API) getEdge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "edgeUuid")
	f, err := a.G.GetEntityEdge(r.Context(), id)
	if err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	a.json(w, http.StatusOK, map[string]any{
		"uuid": f.UUID, "name": f.Name, "fact": f.Fact, "created_at": f.CreatedAt,
		"source_node_uuid": "", "target_node_uuid": "",
	})
}

func (a *API) patchEdge(w http.ResponseWriter, r *http.Request) {
	a.getEdge(w, r)
}

func (a *API) deleteEdge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "edgeUuid")
	_ = a.G.DeleteEntityEdge(r.Context(), id)
	a.json(w, http.StatusOK, map[string]any{"message": "ok"})
}

func (a *API) getEpisodesByGraph(w http.ResponseWriter, r *http.Request) {
	gid := chi.URLParam(r, "graphId")
	lastN := 20
	if v := r.URL.Query().Get("lastn"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			lastN = n
		}
	}
	raw, err := a.G.GetEpisodes(r.Context(), gid, lastN)
	if err != nil {
		a.err(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (a *API) getEpisodesByUser(w http.ResponseWriter, r *http.Request) {
	uid := chi.URLParam(r, "userId")
	lastN := 20
	if v := r.URL.Query().Get("lastn"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			lastN = n
		}
	}
	raw, err := a.G.GetEpisodes(r.Context(), uid, lastN)
	if err != nil {
		a.err(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (a *API) getEpisode(w http.ResponseWriter, r *http.Request) {
	a.json(w, http.StatusOK, map[string]any{"uuid": chi.URLParam(r, "episodeUuid")})
}

func (a *API) deleteEpisode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "episodeUuid")
	_ = a.G.DeleteEpisode(r.Context(), id)
	a.json(w, http.StatusOK, map[string]any{"message": "ok"})
}

func graphToJSON(g *store.GraphRecord) map[string]any {
	m := map[string]any{
		"graph_id": g.GraphID, "metadata": g.Metadata, "created_at": ts(g.CreatedAt),
		"project_uuid": g.ProjectUUID.String(), "uuid": g.UUID.String(),
	}
	if g.UserID != nil {
		m["user_id"] = *g.UserID
	}
	return m
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func firstString(vals ...any) string {
	for _, v := range vals {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return ""
}
