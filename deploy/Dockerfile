FROM python:3.11-slim-bookworm

COPY requirements.txt /tmp/requirements.txt

RUN pip install -r /tmp/requirements.txt

COPY deployment.py /app/deployment.py

ENTRYPOINT ["python", "/app/deployment.py"]