from pydantic import BaseModel, Field

from graph_service.dto.common import Message


class AddMessagesRequest(BaseModel):
    group_id: str = Field(..., description='The group id of the messages to add')
    messages: list[Message] = Field(..., description='The messages to add')


class AddEntityNodeRequest(BaseModel):
    uuid: str = Field(..., description='The uuid of the node to add')
    group_id: str = Field(..., description='The group id of the node to add')
    name: str = Field(..., description='The name of the node to add')
    summary: str = Field(default='', description='The summary of the node to add')


class AddFactTripleRequest(BaseModel):
    subject: str = Field(..., description='The subject entity name')
    predicate: str = Field(..., description='The relationship/predicate name')
    object: str = Field(..., description='The object entity name')
    group_id: str = Field(..., description='The group id for the triple')
    fact: str = Field(default='', description='Human-readable fact sentence')
