# open-context fastapi example

This service uses the **zep-python** SDK (`zep-cloud`) pointed at your **Open Context** Go backend (same `/api/v2` routes as Zep Cloud).

## environment

- `ZEP_API_URL` — host only, e.g. `http://localhost:8000` (SDK appends `/api/v2`).
- `ZEP_API_KEY` — must match `OPEN_CONTEXT_API_KEY` on the backend.
- Or set `OPEN_CONTEXT_API_KEY` only; the example falls back to it.

## run locally

```bash
cd examples/fastapi-app
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
export ZEP_API_URL=http://localhost:8000
export ZEP_API_KEY=changeme
uvicorn main:app --reload --port 8080
```

Try:

1. `POST /demo/user` with `{"user_id":"u1","email":"a@b.com"}`
2. `POST /demo/thread` with `{"thread_id":"t1","user_id":"u1"}`
3. `POST /demo/thread/t1/message` with `{"content":"hello"}`
4. `GET /demo/thread/t1/context`
