import logging
from collections.abc import Awaitable, Callable
from typing import Annotated

from fastapi import Depends, HTTPException
from graphiti_core import Graphiti  # type: ignore
from graphiti_core.driver.driver import GraphProvider
from graphiti_core.edges import EntityEdge  # type: ignore
from graphiti_core.errors import EdgeNotFoundError, GroupsEdgesNotFoundError, NodeNotFoundError
from graphiti_core.llm_client import LLMClient  # type: ignore
from graphiti_core.models.nodes import node_db_queries
from graphiti_core.nodes import EntityNode, EpisodicNode  # type: ignore
from graphiti_core.utils import bulk_utils

from graph_service.config import ZepEnvDep
from graph_service.dto import FactResult

logger = logging.getLogger(__name__)

_original_get_entity_node_save_bulk_query = node_db_queries.get_entity_node_save_bulk_query


def _patched_get_entity_node_save_bulk_query(
    provider: GraphProvider, nodes=None, has_aoss: bool = False
):
    if provider != GraphProvider.NEO4J:
        return _original_get_entity_node_save_bulk_query(provider, nodes, has_aoss)

    save_embedding_query = (
        'WITH n, node CALL db.create.setNodeVectorProperty(n, "name_embedding", node.name_embedding)'
        if not has_aoss
        else ''
    )

    return (
        """
            UNWIND $nodes AS node
            MERGE (n:Entity {uuid: node.uuid})
            SET n = node
            REMOVE n.labels
            """
        + ('REMOVE n.name_embedding\n' if not has_aoss else '')
        + save_embedding_query
        + """
            RETURN n.uuid AS uuid
        """
    )


# graphiti-core 0.28.x uses SET n:$(node.labels), which is rejected by the
# Neo4j version in open-context's compose stack. Patch both import sites.
node_db_queries.get_entity_node_save_bulk_query = _patched_get_entity_node_save_bulk_query
bulk_utils.get_entity_node_save_bulk_query = _patched_get_entity_node_save_bulk_query


class ZepGraphiti(Graphiti):
    def __init__(self, uri: str, user: str, password: str, llm_client: LLMClient | None = None):
        super().__init__(uri, user, password, llm_client)

    async def _run_with_partition_group_ids(self, fn: Callable[[], Awaitable]):
        original_clone = getattr(self.driver, 'clone', None)

        # graphiti-core can treat group_id as a physical Neo4j database name.
        # open-context uses group_id only as a logical partition inside the same
        # Neo4j database, so keep all queries pinned to the existing driver.
        self.driver.clone = lambda database=None: self.driver  # type: ignore[method-assign]

        try:
            return await fn()
        finally:
            if original_clone is not None:
                self.driver.clone = original_clone  # type: ignore[method-assign]

    async def add_episode(self, *args, **kwargs):
        return await self._run_with_partition_group_ids(
            lambda: Graphiti.add_episode(self, *args, **kwargs)
        )

    async def add_episode_bulk(self, *args, **kwargs):
        return await self._run_with_partition_group_ids(
            lambda: Graphiti.add_episode_bulk(self, *args, **kwargs)
        )

    async def save_entity_node(self, name: str, uuid: str, group_id: str, summary: str = ''):
        new_node = EntityNode(
            name=name,
            uuid=uuid,
            group_id=group_id,
            summary=summary,
        )
        await new_node.generate_name_embedding(self.embedder)
        await new_node.save(self.driver)
        return new_node

    async def get_entity_edge(self, uuid: str):
        try:
            edge = await EntityEdge.get_by_uuid(self.driver, uuid)
            return edge
        except EdgeNotFoundError as e:
            raise HTTPException(status_code=404, detail=e.message) from e

    async def delete_group(self, group_id: str):
        try:
            edges = await EntityEdge.get_by_group_ids(self.driver, [group_id])
        except GroupsEdgesNotFoundError:
            logger.warning(f'No edges found for group {group_id}')
            edges = []

        nodes = await EntityNode.get_by_group_ids(self.driver, [group_id])

        episodes = await EpisodicNode.get_by_group_ids(self.driver, [group_id])

        for edge in edges:
            await edge.delete(self.driver)

        for node in nodes:
            await node.delete(self.driver)

        for episode in episodes:
            await episode.delete(self.driver)

    async def delete_entity_edge(self, uuid: str):
        try:
            edge = await EntityEdge.get_by_uuid(self.driver, uuid)
            await edge.delete(self.driver)
        except EdgeNotFoundError as e:
            raise HTTPException(status_code=404, detail=e.message) from e

    async def delete_episodic_node(self, uuid: str):
        try:
            episode = await EpisodicNode.get_by_uuid(self.driver, uuid)
            await episode.delete(self.driver)
        except NodeNotFoundError as e:
            raise HTTPException(status_code=404, detail=e.message) from e


async def get_graphiti(settings: ZepEnvDep):
    client = ZepGraphiti(
        uri=settings.neo4j_uri,
        user=settings.neo4j_user,
        password=settings.neo4j_password,
    )
    if settings.openai_base_url is not None:
        client.llm_client.config.base_url = settings.openai_base_url
    if settings.openai_api_key is not None:
        client.llm_client.config.api_key = settings.openai_api_key
    if settings.model_name is not None:
        client.llm_client.model = settings.model_name

    try:
        yield client
    finally:
        await client.close()


async def initialize_graphiti(settings: ZepEnvDep):
    client = ZepGraphiti(
        uri=settings.neo4j_uri,
        user=settings.neo4j_user,
        password=settings.neo4j_password,
    )
    await client.build_indices_and_constraints()


def get_fact_result_from_edge(edge: EntityEdge):
    return FactResult(
        uuid=edge.uuid,
        name=edge.name,
        fact=edge.fact,
        valid_at=edge.valid_at,
        invalid_at=edge.invalid_at,
        created_at=edge.created_at,
        expired_at=edge.expired_at,
        source_node_uuid=edge.source_node_uuid,
        target_node_uuid=edge.target_node_uuid,
        score=getattr(edge, 'score', None),
        relevance=getattr(edge, 'relevance', None),
        attributes=getattr(edge, 'attributes', None),
    )


ZepGraphitiDep = Annotated[ZepGraphiti, Depends(get_graphiti)]
