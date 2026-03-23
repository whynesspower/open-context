from datetime import datetime, timezone
from enum import Enum

from pydantic import BaseModel, Field

from graph_service.dto.common import Message


class SearchScope(str, Enum):
    edges = 'edges'
    nodes = 'nodes'
    episodes = 'episodes'


class SearchReranker(str, Enum):
    rrf = 'rrf'
    mmr = 'mmr'
    node_distance = 'node_distance'
    episode_mentions = 'episode_mentions'
    cross_encoder = 'cross_encoder'


class SearchQuery(BaseModel):
    group_ids: list[str] | None = Field(
        None, description='The group ids for the memories to search'
    )
    query: str
    max_facts: int = Field(default=10, description='The maximum number of facts to retrieve')
    scope: SearchScope | None = Field(
        default=None,
        description='Which graph layer to search: edges (default), nodes, or episodes',
    )
    reranker: SearchReranker | None = Field(
        default=None, description='Reranking strategy to apply to results'
    )
    mmr_lambda: float | None = Field(
        default=None, description='MMR lambda for diversity vs relevance (0.0-1.0)'
    )
    center_node_uuid: str | None = Field(
        default=None, description='UUID of node to center proximity-based reranking on'
    )
    bfs_origin_node_uuids: list[str] | None = Field(
        default=None, description='Starting node UUIDs for breadth-first search'
    )


class FactResult(BaseModel):
    uuid: str
    name: str
    fact: str
    valid_at: datetime | None
    invalid_at: datetime | None
    created_at: datetime
    expired_at: datetime | None
    source_node_uuid: str | None = None
    target_node_uuid: str | None = None
    score: float | None = None
    relevance: float | None = None
    attributes: dict | None = None

    class Config:
        json_encoders = {datetime: lambda v: v.astimezone(timezone.utc).isoformat()}


class EpisodeResult(BaseModel):
    uuid: str
    name: str | None = None
    group_id: str | None = None
    source: str | None = None
    source_description: str | None = None
    content: str | None = None
    created_at: datetime | None = None

    class Config:
        json_encoders = {datetime: lambda v: v.astimezone(timezone.utc).isoformat()}


class SearchResults(BaseModel):
    facts: list[FactResult] = Field(default_factory=list)
    nodes: list[NodeResult] = Field(default_factory=list)
    episodes: list[EpisodeResult] = Field(default_factory=list)


class GetMemoryRequest(BaseModel):
    group_id: str = Field(..., description='The group id of the memory to get')
    max_facts: int = Field(default=10, description='The maximum number of facts to retrieve')
    center_node_uuid: str | None = Field(
        default=None, description='The uuid of the node to center the retrieval on'
    )
    messages: list[Message] = Field(
        ..., description='The messages to build the retrieval query from '
    )


class GetMemoryResponse(BaseModel):
    facts: list[FactResult] = Field(..., description='The facts that were retrieved from the graph')


class NodeResult(BaseModel):
    uuid: str
    name: str
    summary: str
    labels: list[str] | None = None
    group_id: str | None = None
    created_at: datetime | None = None
    score: float | None = None
    relevance: float | None = None
    attributes: dict | None = None

    class Config:
        json_encoders = {datetime: lambda v: v.astimezone(timezone.utc).isoformat()}


class EdgeResult(BaseModel):
    uuid: str
    name: str
    fact: str
    source_node_uuid: str
    target_node_uuid: str
    valid_at: datetime | None = None
    invalid_at: datetime | None = None
    created_at: datetime | None = None
    expired_at: datetime | None = None
    episodes: list[str] | None = None

    class Config:
        json_encoders = {datetime: lambda v: v.astimezone(timezone.utc).isoformat()}
