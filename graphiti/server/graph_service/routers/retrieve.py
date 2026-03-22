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


@router.get('/episode/{uuid}', status_code=status.HTTP_200_OK)
async def get_episode_by_uuid(uuid: str, graphiti: ZepGraphitiDep):
    from graphiti_core.errors import NodeNotFoundError
    from graphiti_core.nodes import EpisodicNode

    try:
        episode = await EpisodicNode.get_by_uuid(graphiti.driver, uuid)
    except NodeNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e)) from e
    return {
        'uuid': episode.uuid,
        'name': episode.name,
        'group_id': episode.group_id,
        'source': episode.source.value if episode.source else '',
        'source_description': episode.source_description,
        'content': episode.content,
        'created_at': episode.created_at.isoformat() if episode.created_at else None,
    }


@router.get('/episode/{uuid}/mentions', status_code=status.HTTP_200_OK)
async def get_episode_mentions(uuid: str, graphiti: ZepGraphitiDep):
    records, _, _ = await graphiti.driver.execute_query(
        """
        MATCH (ep:Episodic {uuid: $uuid})-[:MENTIONS]->(n:Entity)
        RETURN n.uuid AS uuid, n.name AS name, n.summary AS summary,
               n.group_id AS group_id, n.created_at AS created_at,
               labels(n) AS labels
        """,
        uuid=uuid,
        routing_='r',
    )
    nodes: list[NodeResult] = []
    for r in records:
        raw_labels = r.get('labels', [])
        labels = [lbl for lbl in (raw_labels or []) if lbl != 'Entity']
        nodes.append(
            NodeResult(
                uuid=r['uuid'],
                name=r.get('name', ''),
                summary=r.get('summary', ''),
                labels=labels or None,
                group_id=r.get('group_id'),
                created_at=r.get('created_at'),
            )
        )

    edge_records, _, _ = await graphiti.driver.execute_query(
        """
        MATCH (ep:Episodic {uuid: $uuid})-[:MENTIONS]->(n:Entity)
        WITH collect(n.uuid) AS node_uuids
        MATCH (s:Entity)-[e:RELATES_TO]->(t:Entity)
        WHERE e.uuid IS NOT NULL AND (s.uuid IN node_uuids OR t.uuid IN node_uuids)
              AND $uuid IN e.episodes
        RETURN DISTINCT e.uuid AS uuid, e.name AS name, e.fact AS fact,
               s.uuid AS source_node_uuid, t.uuid AS target_node_uuid,
               e.created_at AS created_at, e.valid_at AS valid_at,
               e.invalid_at AS invalid_at, e.expired_at AS expired_at
        """,
        uuid=uuid,
        routing_='r',
    )
    edges: list[EdgeResult] = []
    for r in edge_records:
        edges.append(
            EdgeResult(
                uuid=r['uuid'],
                name=r.get('name', ''),
                fact=r.get('fact', ''),
                source_node_uuid=r.get('source_node_uuid', ''),
                target_node_uuid=r.get('target_node_uuid', ''),
                created_at=r.get('created_at'),
                valid_at=r.get('valid_at'),
                invalid_at=r.get('invalid_at'),
                expired_at=r.get('expired_at'),
            )
        )
    return {'nodes': nodes, 'edges': edges}


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


@router.get('/node/{uuid}/episodes', status_code=status.HTTP_200_OK)
async def get_node_episodes(uuid: str, graphiti: ZepGraphitiDep):
    records, _, _ = await graphiti.driver.execute_query(
        """
        MATCH (ep:Episodic)-[:MENTIONS]->(n:Entity {uuid: $uuid})
        RETURN ep.uuid AS uuid, ep.name AS name, ep.group_id AS group_id,
               ep.source AS source, ep.source_description AS source_description,
               ep.content AS content, ep.created_at AS created_at
        """,
        uuid=uuid,
        routing_='r',
    )
    episodes = []
    for r in records:
        episodes.append({
            'uuid': r['uuid'],
            'name': r.get('name', ''),
            'group_id': r.get('group_id', ''),
            'source': r.get('source', ''),
            'source_description': r.get('source_description', ''),
            'content': r.get('content', ''),
            'created_at': r.get('created_at'),
        })
    return {'episodes': episodes}


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
