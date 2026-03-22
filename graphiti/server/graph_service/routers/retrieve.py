from datetime import datetime, timezone

from fastapi import APIRouter, status

from graphiti_core.edges import EntityEdge  # type: ignore
from graphiti_core.nodes import EntityNode  # type: ignore

from fastapi import HTTPException

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


@router.patch('/node/{uuid}', status_code=status.HTTP_200_OK)
async def update_node(uuid: str, request: dict, graphiti: ZepGraphitiDep):
    from graphiti_core.errors import NodeNotFoundError

    try:
        node = await EntityNode.get_by_uuid(graphiti.driver, uuid)
    except NodeNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e)) from e
    if 'name' in request:
        node.name = request['name']
        await node.generate_name_embedding(graphiti.embedder)
    if 'summary' in request:
        node.summary = request['summary']
    if 'labels' in request:
        node.labels = request['labels']
    await node.save(graphiti.driver)
    labels = list(node.labels) if node.labels else None
    return NodeResult(
        uuid=node.uuid,
        name=node.name,
        summary=node.summary or '',
        labels=labels,
        group_id=node.group_id,
        created_at=node.created_at,
    )


@router.patch('/entity-edge/{uuid}', status_code=status.HTTP_200_OK)
async def update_entity_edge(uuid: str, request: dict, graphiti: ZepGraphitiDep):
    edge = await graphiti.get_entity_edge(uuid)
    if 'fact' in request:
        edge.fact = request['fact']
    if 'name' in request:
        edge.name = request['name']
    await edge.save(graphiti.driver)
    return get_fact_result_from_edge(edge)


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


@router.get('/node/{uuid}', status_code=status.HTTP_200_OK)
async def get_node(uuid: str, graphiti: ZepGraphitiDep):
    from graphiti_core.errors import NodeNotFoundError

    try:
        node = await EntityNode.get_by_uuid(graphiti.driver, uuid)
    except NodeNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e)) from e
    labels = list(node.labels) if node.labels else None
    return NodeResult(
        uuid=node.uuid,
        name=node.name,
        summary=node.summary or '',
        labels=labels,
        group_id=node.group_id,
        created_at=node.created_at,
    )


@router.get('/node/{uuid}/edges', status_code=status.HTTP_200_OK)
async def get_node_edges(uuid: str, graphiti: ZepGraphitiDep):
    from graphiti_core.errors import EdgeNotFoundError

    try:
        edges = await EntityEdge.get_by_node_uuid(graphiti.driver, uuid)
    except EdgeNotFoundError:
        edges = []
    out: list[EdgeResult] = []
    for e in edges:
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
