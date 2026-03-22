FROM python:3.11-slim
WORKDIR /app
RUN pip install --no-cache-dir uv
COPY pyproject.toml README.md ./
COPY graph_service ./graph_service
RUN pip install --no-cache-dir .
ENV PYTHONUNBUFFERED=1
EXPOSE 8003
CMD ["uvicorn", "graph_service.main:app", "--host", "0.0.0.0", "--port", "8003"]
