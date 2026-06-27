FROM python:3.12-slim AS deps

WORKDIR /build

COPY requirements.txt .

RUN pip install --target=/deps --no-cache-dir -r requirements.txt

FROM python:3.12-slim

WORKDIR /app

RUN apt-get update && \
    apt-get install -y libpq5 && \
    rm -rf /var/lib/apt/lists/*

COPY --from=deps /deps /deps
ENV PYTHONPATH=/deps


COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 8080
CMD ["python", "main.py"]