FROM python:3.11-slim
WORKDIR /app
COPY sdks/zep-python /sdks/zep-python
RUN pip install --no-cache-dir fastapi "uvicorn[standard]" && pip install --no-cache-dir -e /sdks/zep-python
COPY examples/fastapi-app/main.py ./
EXPOSE 8080
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8080"]
