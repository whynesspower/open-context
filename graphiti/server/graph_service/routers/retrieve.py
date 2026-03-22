from datetime import datetime, timezone

from fastapi import APIRouter, status

from graphiti_core.edges import EntityEdge  # type: ignore
from graphiti_core.nodes import EntityNode  # type: ignore

from graph_service.dto import (
    EdgeResult,
    GetMemoryRequest,
    GetMemoryResponse,
    Message,
    NodeResult,
    SearchQuery,
    SearchResults,
)
from graph_service.zep_graphiti import ZepGraphitiDep, get_fact_result_from_edge

router = APIRouter()


@router.post('/search', status_code=status.HTTP_200_OK)
async def search(query: SearchQuery, graphiti: ZepGraphitiDep):
    relevant_edges = await graphiti.search(
        group_ids=query.group_ids,
        query=query.query,
        num_results=query.max_facts,
    )
    facts = [get_fact_result_from_edge(edge) for edge in relevant_edges]
    return SearchResults(
        facts=facts,
    )


@router.get('/entity-edge/{uuid}', status_code=status.HTTP_200_OK)
async def get_entity_edge(uuid: str, graphiti: ZepGraphitiDep):
    entity_edge = await graphiti.get_entity_edge(uuid)
    return get_fact_result_from_edge(entity_edge)


@router.get('/episodes/{group_id}', status_code=status.HTTP_200_OK)
async def get_episodes(group_id: str, last_n: int, graphiti: ZepGraphitiDep):
    episodes = await graphiti.retrieve_episodes(
        group_ids=[group_id], last_n=last_n, reference_time=datetime.now(timezone.utc)
    )
    return episodes


@router.post('/get-memory', status_code=status.HTTP_200_OK)
async def get_memory(
    request: GetMemoryRequest,
    graphiti: ZepGraphitiDep,
):
    combined_query = compose_query_from_messages(request.messages)
    result = await graphiti.search(
        group_ids=[request.group_id],
        query=combined_query,
        num_results=request.max_facts,
    )
    facts = [get_fact_result_from_edge(edge) for edge in result]
    return GetMemoryResponse(facts=facts)


def compose_query_from_messages(messages: list[Message]):
    combined_query = ''
    for message in messages:
        combined_query += f'{message.role_type or ""}({message.role or ""}): {message.content}\n'
    return combined_query


@router.get('/nodes/{group_id}', status_code=status.HTTP_200_OK)
async def list_nodes(group_id: str, graphiti: ZepGraphitiDep, limit: int = 500):
    cap = max(1, min(limit, 500))
    nodes = await EntityNode.get_by_group_ids(graphiti.driver, [group_id])
    out: list[NodeResult] = []
    for n in nodes[:cap]:
        labels = list(n.labels) if n.labels else None
        out.append(
            NodeResult(
                uuid=n.uuid,
                name=n.name,
                summary=n.summary or '',
                labels=labels,
                group_id=n.group_id,
                created_at=n.created_at,
            )
        )
    return out


@router.get('/edges/{group_id}', status_code=status.HTTP_200_OK)
async def list_edges(group_id: str, graphiti: ZepGraphitiDep, limit: int = 500):
    cap = max(1, min(limit, 500))
    edges = await EntityEdge.get_by_group_ids(graphiti.driver, [group_id])
    out: list[EdgeResult] = []
    for e in edges[:cap]:
        out.append(
            EdgeResult(
                uuid=e.uuid,
                name=e.name,
                fact=e.fact,
                source_node_uuid=e.source_node_uuid,
                target_node_uuid=e.target_node_uuid,
                valid_at=e.valid_at,
                invalid_at=e.invalid_at,
                created_at=e.created_at,
                expired_at=e.expired_at,
                episodes=list(e.episodes) if e.episodes else None,
            )
        )
    return out
