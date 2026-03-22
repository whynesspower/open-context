"""Minimal FastAPI demo using zep-python against Open Context backend."""

import os

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from zep_cloud import Zep
from zep_cloud.types.message import Message

app = FastAPI(title="open-context example", version="0.1.0")


def client() -> Zep:
    api_key = os.environ.get("ZEP_API_KEY") or os.environ.get("OPEN_CONTEXT_API_KEY", "")
    if not api_key:
        raise HTTPException(500, "set ZEP_API_KEY or OPEN_CONTEXT_API_KEY")
    return Zep(api_key=api_key)


class UserIn(BaseModel):
    user_id: str
    email: str | None = None


class ThreadIn(BaseModel):
    thread_id: str
    user_id: str


class MsgIn(BaseModel):
    content: str
    role: str = "user"


@app.get("/health")
def health():
    return {"status": "ok"}


@app.post("/demo/user")
def demo_user(body: UserIn):
    z = client()
    u = z.user.add(
        user_id=body.user_id,
        email=body.email,
    )
    return {"user": u.model_dump()}


@app.post("/demo/thread")
def demo_thread(body: ThreadIn):
    z = client()
    t = z.thread.create(thread_id=body.thread_id, user_id=body.user_id)
    return {"thread": t.model_dump()}


@app.post("/demo/thread/{thread_id}/message")
def demo_message(thread_id: str, body: MsgIn):
    z = client()
    role: str = "user"
    if body.role in ("assistant", "system", "norole", "function", "tool"):
        role = body.role
    resp = z.thread.add_messages(
        thread_id=thread_id,
        messages=[Message(content=body.content, role=role)],
    )
    return {"response": resp.model_dump()}


@app.get("/demo/thread/{thread_id}/context")
def demo_context(thread_id: str):
    z = client()
    ctx = z.thread.get_user_context(thread_id=thread_id)
    return {"context": ctx.model_dump()}
