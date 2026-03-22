package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (a *API) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
	}))
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/v2", func(r chi.Router) {
		r.Use(a.Auth)

		r.Get("/projects/info", a.projectInfo)
		r.Get("/tasks/{taskId}", a.getTask)

		r.Get("/context-templates", a.listContextTemplates)
		r.Post("/context-templates", a.createContextTemplate)
		r.Get("/context-templates/{templateId}", a.getContextTemplate)
		r.Put("/context-templates/{templateId}", a.updateContextTemplate)
		r.Delete("/context-templates/{templateId}", a.deleteContextTemplate)

		r.Get("/custom-instructions", a.listCustomInstructions)
		r.Post("/custom-instructions", a.addCustomInstructions)
		r.Delete("/custom-instructions", a.deleteCustomInstructions)

		r.Get("/entity-types", a.listEntityTypes)
		r.Put("/entity-types", a.putEntityTypes)

		r.Get("/user-summary-instructions", a.listUserSummaryInstructions)
		r.Post("/user-summary-instructions", a.addUserSummaryInstructions)
		r.Delete("/user-summary-instructions", a.deleteUserSummaryInstructions)

		r.Post("/users", a.postUsers)
		r.Get("/users-ordered", a.getUsersOrdered)
		r.Get("/users/{userId}", a.getUser)
		r.Patch("/users/{userId}", a.patchUser)
		r.Delete("/users/{userId}", a.deleteUser)
		r.Get("/users/{userId}/threads", a.getUserThreads)
		r.Get("/users/{userId}/node", a.getUserNode)
		r.Get("/users/{userId}/warm", a.warmUser)

		r.Get("/threads", a.listThreads)
		r.Post("/threads", a.createThread)
		r.Delete("/threads/{threadId}", a.deleteThread)
		r.Get("/threads/{threadId}/messages", a.getThreadMessages)
		r.Post("/threads/{threadId}/messages", a.addThreadMessages)
		r.Post("/threads/{threadId}/messages-batch", a.addThreadMessagesBatch)
		r.Get("/threads/{threadId}/context", a.getThreadContext)

		r.Patch("/messages/{messageUuid}", a.patchMessage)

		r.Post("/graph/search", a.graphSearch)
		r.Post("/graph/create", a.graphCreate)
		r.Get("/graph/list-all", a.graphListAll)
		r.Get("/graph/{graphId}", a.graphGet)
		r.Patch("/graph/{graphId}", a.graphPatch)
		r.Delete("/graph/{graphId}", a.graphDelete)
		r.Post("/graph", a.graphAdd)
		r.Post("/graph-batch", a.graphAddBatch)
		r.Post("/graph/add-fact-triple", a.graphAddFactTriple)
		r.Post("/graph/clone", a.graphClone)
		r.Post("/graph/patterns", a.graphPatterns)

		r.Post("/graph/node/graph/{graphId}", a.postNodesByGraph)
		r.Post("/graph/node/user/{userId}", a.postNodesByUser)
		r.Get("/graph/node/{nodeUuid}", a.getNode)
		r.Patch("/graph/node/{nodeUuid}", a.patchNode)
		r.Delete("/graph/node/{nodeUuid}", a.deleteNode)
		r.Get("/graph/node/{nodeUuid}/entity-edges", a.getNodeEdges)
		r.Get("/graph/node/{nodeUuid}/episodes", a.getNodeEpisodes)

		r.Post("/graph/edge/graph/{graphId}", a.postEdgesByGraph)
		r.Post("/graph/edge/user/{userId}", a.postEdgesByUser)
		r.Get("/graph/edge/{edgeUuid}", a.getEdge)
		r.Patch("/graph/edge/{edgeUuid}", a.patchEdge)
		r.Delete("/graph/edge/{edgeUuid}", a.deleteEdge)

		r.Get("/graph/episodes/graph/{graphId}", a.getEpisodesByGraph)
		r.Get("/graph/episodes/user/{userId}", a.getEpisodesByUser)
		r.Get("/graph/episodes/{episodeUuid}", a.getEpisode)
		r.Get("/graph/episodes/{episodeUuid}/mentions", a.getEpisodeMentions)
		r.Delete("/graph/episodes/{episodeUuid}", a.deleteEpisode)
	})

	return r
}
