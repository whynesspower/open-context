package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

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
	if v, ok := body["max_facts"].(float64); ok {
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
	a.json(w, http.StatusOK, map[string]any{"edges": edges, "nodes": []any{}, "episodes": []any{}})
}

func factToEdge(f graphiti.FactResult) map[string]any {
	return map[string]any{
		"uuid": f.UUID, "name": f.Name, "fact": f.Fact,
		"created_at": f.CreatedAt, "valid_at": f.ValidAt, "invalid_at": f.InvalidAt, "expired_at": f.ExpiredAt,
		"source_node_uuid": f.SourceNodeUUID, "target_node_uuid": f.TargetNodeUUID,
		"score": f.Score, "relevance": f.Relevance, "attributes": f.Attributes, "episodes": []any{},
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
	name, _ := body["name"].(string)
	if name == "" {
		name = gid
	}
	desc, _ := body["description"].(string)
	rec := &store.GraphRecord{
		GraphID:     gid,
		Name:        name,
		Description: desc,
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
	total, _ := a.DB.NewSelect().Model((*store.GraphRecord)(nil)).Where("project_uuid = ?", a.DB.Project).Count(r.Context())
	_ = a.DB.NewSelect().Model(&graphs).Where("project_uuid = ?", a.DB.Project).Order("created_at DESC").Limit(500).Scan(r.Context())
	out := make([]any, 0, len(graphs))
	for i := range graphs {
		out = append(out, graphToJSON(&graphs[i]))
	}
	a.json(w, http.StatusOK, map[string]any{"graphs": out, "row_count": len(out), "total_count": total})
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
	if v, ok := body["name"].(string); ok && v != "" {
		g.Name = v
	}
	if v, ok := body["description"].(string); ok {
		g.Description = v
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
	var items []map[string]any
	if err := a.readJSON(r, &items); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body: expected array")
		return
	}

	taskID := uuid.NewString()
	task := &store.TaskRecord{
		TaskID:      taskID,
		Status:      "pending",
		Progress:    0,
		ProjectUUID: a.DB.Project,
		CreatedAt:   a.now(),
		UpdatedAt:   a.now(),
	}
	_, _ = a.DB.NewInsert().Model(task).Exec(r.Context())

	// Process batch asynchronously so caller can poll task status
	go func(items []map[string]any, taskID string) {
		ctx := context.Background()
		total := len(items)
		processed := 0
		var lastErr error
		for _, item := range items {
			group := firstString(item["graph_id"], item["user_id"])
			if group == "" {
				processed++
				continue
			}
			data, _ := item["data"].(string)
			typ, _ := item["type"].(string)
			gm := graphiti.GMessage{Content: data, RoleType: "user", Role: typ, UUID: uuid.NewString()}
			if err := a.G.AddMessages(ctx, group, []graphiti.GMessage{gm}); err != nil {
				lastErr = err
			}
			processed++
			progress := float64(processed) / float64(total)
			_, _ = a.DB.NewUpdate().Model(&store.TaskRecord{}).
				Set("status = ?, progress = ?, updated_at = ?", "processing", progress, time.Now().UTC()).
				Where("task_id = ?", taskID).
				Exec(ctx)
		}
		status := "completed"
		errMsg := ""
		if lastErr != nil {
			status = "failed"
			errMsg = lastErr.Error()
		}
		_, _ = a.DB.NewUpdate().Model(&store.TaskRecord{}).
			Set("status = ?, progress = ?, error = ?, updated_at = ?", status, 1.0, errMsg, time.Now().UTC()).
			Where("task_id = ?", taskID).
			Exec(ctx)
	}(items, taskID)

	a.json(w, http.StatusOK, map[string]any{"task_id": taskID, "status": "pending"})
}

func (a *API) graphAddFactTriple(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Subject   string `json:"subject_node_name"`
		Predicate string `json:"predicate_name"`
		Object    string `json:"object_node_name"`
		GraphID   string `json:"graph_id"`
		UserID    string `json:"user_id"`
		Fact      string `json:"fact"`
	}
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	group := body.GraphID
	if group == "" {
		group = body.UserID
	}
	if group == "" || body.Subject == "" || body.Predicate == "" || body.Object == "" {
		a.err(w, http.StatusBadRequest, "subject_node_name, predicate_name, object_node_name, and graph_id or user_id required")
		return
	}
	result, err := a.G.AddFactTriple(r.Context(), graphiti.AddFactTripleRequest{
		Subject:   body.Subject,
		Predicate: body.Predicate,
		Object:    body.Object,
		GroupID:   group,
		Fact:      body.Fact,
	})
	if err != nil {
		a.err(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.json(w, http.StatusOK, factToEdge(*result))
}

func (a *API) graphClone(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	srcGraphID, _ := body["source_graph_id"].(string)
	dstGraphID, _ := body["new_graph_id"].(string)
	if srcGraphID == "" {
		a.err(w, http.StatusBadRequest, "source_graph_id required")
		return
	}
	if dstGraphID == "" {
		dstGraphID = uuid.NewString()
	}

	var srcGraph store.GraphRecord
	if err := a.DB.NewSelect().Model(&srcGraph).Where("graph_id = ? AND project_uuid = ?", srcGraphID, a.DB.Project).Scan(r.Context()); err != nil {
		a.err(w, http.StatusNotFound, "source graph not found")
		return
	}

	newRec := &store.GraphRecord{
		GraphID:     dstGraphID,
		Name:        srcGraph.Name,
		Description: srcGraph.Description,
		ProjectUUID: a.DB.Project,
		Metadata:    srcGraph.Metadata,
		UserID:      srcGraph.UserID,
	}
	if _, err := a.DB.NewInsert().Model(newRec).Exec(r.Context()); err != nil {
		a.err(w, http.StatusBadRequest, "target graph already exists")
		return
	}

	// Clone nodes first, building old→new UUID map for edge remapping
	nodes, _ := a.G.ListNodes(r.Context(), srcGraphID, 500)
	nodeUUIDMap := make(map[string]string, len(nodes))
	for _, n := range nodes {
		newUUID := uuid.NewString()
		nodeUUIDMap[n.UUID] = newUUID
		_ = a.G.AddEntityNode(r.Context(), graphiti.AddEntityNodeRequest{
			UUID: newUUID, GroupID: dstGraphID, Name: n.Name, Summary: n.Summary,
		})
	}

	// Clone edges, remapping source/target UUIDs to cloned node UUIDs
	edges, _ := a.G.ListEdges(r.Context(), srcGraphID, 500)
	for _, e := range edges {
		newSrc := nodeUUIDMap[e.SourceNodeUUID]
		newTgt := nodeUUIDMap[e.TargetNodeUUID]
		if newSrc == "" || newTgt == "" {
			continue
		}
		_, _ = a.G.AddFactTriple(r.Context(), graphiti.AddFactTripleRequest{
			Subject:   newSrc,
			Predicate: e.Name,
			Object:    newTgt,
			GroupID:   dstGraphID,
			Fact:      e.Fact,
		})
	}

	a.json(w, http.StatusOK, map[string]any{
		"message":         "cloned",
		"source_graph_id": srcGraphID,
		"new_graph_id":    dstGraphID,
		"nodes_cloned":    len(nodes),
		"edges_cloned":    len(edges),
	})
}

func (a *API) graphPatterns(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	graphID := firstString(body["graph_id"], body["user_id"])
	if graphID == "" {
		a.err(w, http.StatusBadRequest, "graph_id or user_id required")
		return
	}

	nodes, err := a.G.ListNodes(r.Context(), graphID, 500)
	if err != nil {
		nodes = nil
	}
	edges, err := a.G.ListEdges(r.Context(), graphID, 500)
	if err != nil {
		edges = nil
	}

	labelCounts := map[string]int{}
	for _, n := range nodes {
		for _, l := range n.Labels {
			labelCounts[l]++
		}
	}
	edgeNameCounts := map[string]int{}
	nodeDegree := map[string]int{}
	for _, e := range edges {
		edgeNameCounts[e.Name]++
		nodeDegree[e.SourceNodeUUID]++
		nodeDegree[e.TargetNodeUUID]++
	}

	topLabels := make([]map[string]any, 0)
	for label, count := range labelCounts {
		topLabels = append(topLabels, map[string]any{"label": label, "count": count})
	}
	topEdgeNames := make([]map[string]any, 0)
	for name, count := range edgeNameCounts {
		topEdgeNames = append(topEdgeNames, map[string]any{"name": name, "count": count})
	}

	var maxDegreeNode string
	maxDegree := 0
	for nodeUUID, deg := range nodeDegree {
		if deg > maxDegree {
			maxDegree = deg
			maxDegreeNode = nodeUUID
		}
	}

	patterns := []map[string]any{
		{"type": "summary", "total_nodes": len(nodes), "total_edges": len(edges)},
		{"type": "label_distribution", "labels": topLabels},
		{"type": "edge_name_distribution", "edge_names": topEdgeNames},
	}
	if maxDegreeNode != "" {
		patterns = append(patterns, map[string]any{
			"type": "highest_degree_node", "node_uuid": maxDegreeNode, "degree": maxDegree,
		})
	}

	a.json(w, http.StatusOK, map[string]any{"patterns": patterns})
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
			"score": nil, "relevance": nil, "attributes": map[string]any{},
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
			"score": nil, "relevance": nil, "attributes": map[string]any{},
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
	n, err := a.G.GetNode(r.Context(), id)
	if err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	a.json(w, http.StatusOK, map[string]any{
		"uuid": n.UUID, "name": n.Name, "summary": n.Summary,
		"labels": n.Labels, "created_at": n.CreatedAt,
		"score": nil, "relevance": nil, "attributes": map[string]any{},
	})
}

func (a *API) patchNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "nodeUuid")
	var body map[string]any
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	n, err := a.G.UpdateNode(r.Context(), id, body)
	if err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	a.json(w, http.StatusOK, map[string]any{
		"uuid": n.UUID, "name": n.Name, "summary": n.Summary,
		"labels": n.Labels, "created_at": n.CreatedAt,
		"score": nil, "relevance": nil, "attributes": map[string]any{},
	})
}

func (a *API) deleteNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "nodeUuid")
	if err := a.G.DeleteNode(r.Context(), id); err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	a.json(w, http.StatusOK, map[string]any{"message": "ok"})
}

func (a *API) getNodeEdges(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "nodeUuid")
	edges, err := a.G.GetNodeEdges(r.Context(), id)
	if err != nil {
		a.json(w, http.StatusOK, []any{})
		return
	}
	a.json(w, http.StatusOK, edgesToSDK(edges))
}

func (a *API) getNodeEpisodes(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "nodeUuid")
	raw, err := a.G.GetNodeEpisodes(r.Context(), id)
	if err != nil {
		a.json(w, http.StatusOK, map[string]any{"episodes": []any{}})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
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
		"valid_at": f.ValidAt, "invalid_at": f.InvalidAt, "expired_at": f.ExpiredAt,
		"source_node_uuid": f.SourceNodeUUID, "target_node_uuid": f.TargetNodeUUID,
		"score": nil, "relevance": nil, "attributes": map[string]any{},
	})
}

func (a *API) patchEdge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "edgeUuid")
	var body map[string]any
	if err := a.readJSON(r, &body); err != nil {
		a.err(w, http.StatusBadRequest, "invalid body")
		return
	}
	f, err := a.G.UpdateEntityEdge(r.Context(), id, body)
	if err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	a.json(w, http.StatusOK, map[string]any{
		"uuid": f.UUID, "name": f.Name, "fact": f.Fact, "created_at": f.CreatedAt,
		"valid_at": f.ValidAt, "invalid_at": f.InvalidAt, "expired_at": f.ExpiredAt,
		"source_node_uuid": f.SourceNodeUUID, "target_node_uuid": f.TargetNodeUUID,
		"score": nil, "relevance": nil, "attributes": map[string]any{},
	})
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
	id := chi.URLParam(r, "episodeUuid")
	raw, err := a.G.GetEpisode(r.Context(), id)
	if err != nil {
		a.err(w, http.StatusNotFound, "not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (a *API) getEpisodeMentions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "episodeUuid")
	raw, err := a.G.GetEpisodeMentions(r.Context(), id)
	if err != nil {
		a.json(w, http.StatusOK, map[string]any{"nodes": []any{}, "edges": []any{}})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (a *API) deleteEpisode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "episodeUuid")
	_ = a.G.DeleteEpisode(r.Context(), id)
	a.json(w, http.StatusOK, map[string]any{"message": "ok"})
}

func graphToJSON(g *store.GraphRecord) map[string]any {
	m := map[string]any{
		"graph_id":     g.GraphID,
		"name":         g.Name,
		"description":  g.Description,
		"metadata":     g.Metadata,
		"created_at":   ts(g.CreatedAt),
		"project_uuid": g.ProjectUUID.String(),
		"uuid":         g.UUID.String(),
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
