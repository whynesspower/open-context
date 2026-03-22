package graphiti

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) (int, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		buf = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+"/"+path, buf)
	if err != nil {
		return 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, err
	}
	if out != nil && len(raw) > 0 && resp.StatusCode < 300 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode: %w body=%s", err, string(raw))
		}
	}
	return resp.StatusCode, nil
}

// --- DTOs matching graphiti server ---

type GMessage struct {
	Content           string `json:"content"`
	UUID              string `json:"uuid,omitempty"`
	Name              string `json:"name"`
	RoleType          string `json:"role_type"`
	Role              string `json:"role,omitempty"`
	Timestamp         string `json:"timestamp,omitempty"`
	SourceDescription string `json:"source_description,omitempty"`
}

type AddMessagesRequest struct {
	GroupID string     `json:"group_id"`
	Messages []GMessage `json:"messages"`
}

type SearchQuery struct {
	GroupIDs []string `json:"group_ids,omitempty"`
	Query    string   `json:"query"`
	MaxFacts int      `json:"max_facts"`
}

type FactResult struct {
	UUID           string  `json:"uuid"`
	Name           string  `json:"name"`
	Fact           string  `json:"fact"`
	ValidAt        *string `json:"valid_at"`
	InvalidAt      *string `json:"invalid_at"`
	CreatedAt      string  `json:"created_at"`
	ExpiredAt      *string `json:"expired_at"`
	SourceNodeUUID string  `json:"source_node_uuid"`
	TargetNodeUUID string  `json:"target_node_uuid"`
}

type SearchResults struct {
	Facts []FactResult `json:"facts"`
}

type GetMemoryRequest struct {
	GroupID        string     `json:"group_id"`
	MaxFacts       int        `json:"max_facts"`
	CenterNodeUUID *string    `json:"center_node_uuid,omitempty"`
	Messages       []GMessage `json:"messages"`
}

type GetMemoryResponse struct {
	Facts []FactResult `json:"facts"`
}

type AddEntityNodeRequest struct {
	UUID    string `json:"uuid"`
	GroupID string `json:"group_id"`
	Name    string `json:"name"`
	Summary string `json:"summary,omitempty"`
}

type GraphitiNode struct {
	UUID      string   `json:"uuid"`
	Name      string   `json:"name"`
	Summary   string   `json:"summary"`
	Labels    []string `json:"labels,omitempty"`
	GroupID   string   `json:"group_id,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

type GraphitiEdge struct {
	UUID             string   `json:"uuid"`
	Name             string   `json:"name"`
	Fact             string   `json:"fact"`
	SourceNodeUUID   string   `json:"source_node_uuid"`
	TargetNodeUUID   string   `json:"target_node_uuid"`
	ValidAt          *string  `json:"valid_at"`
	InvalidAt        *string  `json:"invalid_at"`
	CreatedAt        string   `json:"created_at"`
	ExpiredAt        *string  `json:"expired_at"`
	Episodes         []string `json:"episodes,omitempty"`
}

func (c *Client) AddMessages(ctx context.Context, groupID string, msgs []GMessage) error {
	st, err := c.do(ctx, http.MethodPost, "messages", AddMessagesRequest{GroupID: groupID, Messages: msgs}, nil)
	if err != nil {
		return err
	}
	if st != http.StatusAccepted && st != http.StatusOK {
		return fmt.Errorf("graphiti add messages: status %d", st)
	}
	return nil
}

func (c *Client) Search(ctx context.Context, q SearchQuery) (*SearchResults, error) {
	var out SearchResults
	st, err := c.do(ctx, http.MethodPost, "search", q, &out)
	if err != nil {
		return nil, err
	}
	if st != http.StatusOK {
		return nil, fmt.Errorf("graphiti search: status %d", st)
	}
	return &out, nil
}

func (c *Client) GetMemory(ctx context.Context, req GetMemoryRequest) (*GetMemoryResponse, error) {
	var out GetMemoryResponse
	st, err := c.do(ctx, http.MethodPost, "get-memory", req, &out)
	if err != nil {
		return nil, err
	}
	if st != http.StatusOK {
		return nil, fmt.Errorf("graphiti get-memory: status %d", st)
	}
	return &out, nil
}

type AddFactTripleRequest struct {
	Subject  string `json:"subject"`
	Predicate string `json:"predicate"`
	Object   string `json:"object"`
	GroupID  string `json:"group_id"`
	Fact     string `json:"fact,omitempty"`
}

func (c *Client) AddFactTriple(ctx context.Context, req AddFactTripleRequest) (*FactResult, error) {
	var out FactResult
	st, err := c.do(ctx, http.MethodPost, "add-fact-triple", req, &out)
	if err != nil {
		return nil, err
	}
	if st != http.StatusCreated && st != http.StatusOK {
		return nil, fmt.Errorf("graphiti add-fact-triple: status %d", st)
	}
	return &out, nil
}

func (c *Client) AddEntityNode(ctx context.Context, req AddEntityNodeRequest) error {
	st, err := c.do(ctx, http.MethodPost, "entity-node", req, nil)
	if err != nil {
		return err
	}
	if st != http.StatusCreated && st != http.StatusOK {
		return fmt.Errorf("graphiti entity-node: status %d", st)
	}
	return nil
}

func (c *Client) GetEntityEdge(ctx context.Context, uuid string) (*FactResult, error) {
	var out FactResult
	st, err := c.do(ctx, http.MethodGet, "entity-edge/"+uuid, nil, &out)
	if err != nil {
		return nil, err
	}
	if st != http.StatusOK {
		return nil, fmt.Errorf("graphiti entity-edge: status %d", st)
	}
	return &out, nil
}

func (c *Client) UpdateEntityEdge(ctx context.Context, uuid string, body map[string]any) (*FactResult, error) {
	var out FactResult
	st, err := c.do(ctx, http.MethodPatch, "entity-edge/"+uuid, body, &out)
	if err != nil {
		return nil, err
	}
	if st != http.StatusOK {
		return nil, fmt.Errorf("graphiti update edge: status %d", st)
	}
	return &out, nil
}

func (c *Client) DeleteEntityEdge(ctx context.Context, uuid string) error {
	st, err := c.do(ctx, http.MethodDelete, "entity-edge/"+uuid, nil, nil)
	if err != nil {
		return err
	}
	if st != http.StatusOK && st != http.StatusNoContent && st != http.StatusAccepted {
		return fmt.Errorf("graphiti delete edge: status %d", st)
	}
	return nil
}

func (c *Client) DeleteGroup(ctx context.Context, groupID string) error {
	st, err := c.do(ctx, http.MethodDelete, "group/"+groupID, nil, nil)
	if err != nil {
		return err
	}
	if st != http.StatusOK && st != http.StatusNoContent && st != http.StatusAccepted {
		return fmt.Errorf("graphiti delete group: status %d", st)
	}
	return nil
}

func (c *Client) DeleteEpisode(ctx context.Context, uuid string) error {
	st, err := c.do(ctx, http.MethodDelete, "episode/"+uuid, nil, nil)
	if err != nil {
		return err
	}
	if st != http.StatusOK && st != http.StatusNoContent && st != http.StatusAccepted {
		return fmt.Errorf("graphiti delete episode: status %d", st)
	}
	return nil
}

func (c *Client) GetEpisodes(ctx context.Context, groupID string, lastN int) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/episodes/%s?last_n=%d", c.BaseURL, groupID, lastN), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphiti episodes: status %d %s", resp.StatusCode, string(raw))
	}
	return json.RawMessage(raw), nil
}

func (c *Client) ListNodes(ctx context.Context, groupID string, limit int) ([]GraphitiNode, error) {
	if limit <= 0 {
		limit = 500
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/nodes/%s?limit=%d", c.BaseURL, groupID, limit), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphiti nodes: status %d %s", resp.StatusCode, string(raw))
	}
	var out []GraphitiNode
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListEdges(ctx context.Context, groupID string, limit int) ([]GraphitiEdge, error) {
	if limit <= 0 {
		limit = 500
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/edges/%s?limit=%d", c.BaseURL, groupID, limit), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphiti edges: status %d %s", resp.StatusCode, string(raw))
	}
	var out []GraphitiEdge
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
