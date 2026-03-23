from datetime import datetime, timezone

from fastapi import APIRouter, status

from graphiti_core.edges import EntityEdge  # type: ignore
from graphiti_core.errors import GroupsEdgesNotFoundError  # type: ignore
from graphiti_core.nodes import EntityNode  # type: ignore
from graphiti_core.search.search_config import (  # type: ignore
    EdgeReranker,
    EdgeSearchConfig,
    EdgeSearchMethod,
    EpisodeReranker,
    EpisodeSearchConfig,
    EpisodeSearchMethod,
    NodeReranker,
    NodeSearchConfig,
    NodeSearchMethod,
    SearchConfig,
)

from fastapi import HTTPException

from graph_service.dto import (
    EdgeResult,
    EpisodeResult,
    GetMemoryRequest,
    GetMemoryResponse,
    Message,
    NodeResult,
    SearchQuery,
    SearchResults,
)
from graph_service.dto.retrieve import SearchReranker, SearchScope
from graph_service.zep_graphiti import ZepGraphitiDep, get_fact_result_from_edge

router = APIRouter()


def _map_edge_reranker(r: SearchReranker) -> EdgeReranker:
    return {
        SearchReranker.rrf: EdgeReranker.rrf,
        SearchReranker.mmr: EdgeReranker.mmr,
        SearchReranker.node_distance: EdgeReranker.node_distance,
        SearchReranker.episode_mentions: EdgeReranker.episode_mentions,
        SearchReranker.cross_encoder: EdgeReranker.cross_encoder,
    }[r]


def _map_node_reranker(r: SearchReranker) -> NodeReranker:
    return {
        SearchReranker.rrf: NodeReranker.rrf,
        SearchReranker.mmr: NodeReranker.mmr,
        SearchReranker.node_distance: NodeReranker.node_distance,
        SearchReranker.episode_mentions: NodeReranker.episode_mentions,
        SearchReranker.cross_encoder: NodeReranker.cross_encoder,
    }[r]


def _map_episode_reranker(r: SearchReranker) -> EpisodeReranker:
    mapping: dict[SearchReranker, EpisodeReranker] = {
        SearchReranker.rrf: EpisodeReranker.rrf,
        SearchReranker.cross_encoder: EpisodeReranker.cross_encoder,
    }
    return mapping.get(r, EpisodeReranker.rrf)


def _build_search_config(query: SearchQuery) -> SearchConfig:
    scope = query.scope
    reranker = query.reranker
    mmr_lambda = query.mmr_lambda

    if scope == SearchScope.episodes:
        ep_reranker = _map_episode_reranker(reranker) if reranker else EpisodeReranker.rrf
        ep_config = EpisodeSearchConfig(
            search_methods=[EpisodeSearchMethod.bm25],
            reranker=ep_reranker,
        )
        if mmr_lambda is not None:
            ep_config.mmr_lambda = mmr_lambda
        return SearchConfig(episode_config=ep_config, limit=query.max_facts)

    if scope == SearchScope.nodes:
        n_reranker = _map_node_reranker(reranker) if reranker else NodeReranker.rrf
        n_config = NodeSearchConfig(
            search_methods=[NodeSearchMethod.bm25, NodeSearchMethod.cosine_similarity],
            reranker=n_reranker,
        )
        if mmr_lambda is not None:
            n_config.mmr_lambda = mmr_lambda
        return SearchConfig(node_config=n_config, limit=query.max_facts)

    e_reranker = _map_edge_reranker(reranker) if reranker else EdgeReranker.rrf
    e_config = EdgeSearchConfig(
        search_methods=[EdgeSearchMethod.bm25, EdgeSearchMethod.cosine_similarity],
        reranker=e_reranker,
    )
    if mmr_lambda is not None:
        e_config.mmr_lambda = mmr_lambda
    return SearchConfig(edge_config=e_config, limit=query.max_facts)


@router.post('/search', status_code=status.HTTP_200_OK)
async def search(query: SearchQuery, graphiti: ZepGraphitiDep):
    has_advanced = (
        query.scope is not None
        or query.reranker is not None
        or query.mmr_lambda is not None
        or query.center_node_uuid is not None
        or query.bfs_origin_node_uuids is not None
    )

    if not has_advanced:
        relevant_edges = await graphiti.search(
            group_ids=query.group_ids,
            query=query.query,
            num_results=query.max_facts,
            center_node_uuid=query.center_node_uuid,
        )
        facts = [get_fact_result_from_edge(edge) for edge in relevant_edges]
        return SearchResults(facts=facts)

    config = _build_search_config(query)
    results = await graphiti.search_(
        query=query.query,
        config=config,
        group_ids=query.group_ids,
        center_node_uuid=query.center_node_uuid,
        bfs_origin_node_uuids=query.bfs_origin_node_uuids,
    )

    facts = [get_fact_result_from_edge(edge) for edge in results.edges]
    nodes = [
        NodeResult(
            uuid=n.uuid,
            name=n.name,
            summary=n.summary or '',
            labels=list(n.labels) if n.labels else None,
            group_id=n.group_id,
            created_at=n.created_at,
        )
        for n in results.nodes
    ]
    episodes = [
        EpisodeResult(
            uuid=ep.uuid,
            name=ep.name,
            group_id=ep.group_id,
            source=ep.source.value if ep.source else None,
            source_description=ep.source_description,
            content=ep.content,
            created_at=ep.created_at,
        )
        for ep in results.episodes
    ]
    return SearchResults(facts=facts, nodes=nodes, episodes=episodes)


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
        center_node_uuid=request.center_node_uuid,
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
    try:
        edges = await EntityEdge.get_by_group_ids(graphiti.driver, [group_id])
    except GroupsEdgesNotFoundError:
        edges = []
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
