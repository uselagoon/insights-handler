# Lagoon Insights Handler

This service will listen for messages and handle the requirements of the payload.

## Facts
Currently, the main purpose is to consume a Software Bill of Materials (SBOM) of facts from the logs queue, process
and push to the api and s3 bucket.


## Local development

    go run main.go \
        -rabbitmq-username guest \
        -rabbitmq-password guest \
        -lagoon-api-host http://localhost:7070/graphql \
        --jwt-token-signing-key secret  \
        --access-key-id minio \
        --secret-access-key minio123

To compile GraphQL schema, type-safe structs and response data with genqlient we just add a query/mutation inside of `lagoonclient/genqlient.graphql` and run this:

    go generate

## Configmap labels

```json
  "labels": {
    "lagoon.sh/project": "lagoon",
    "lagoon.sh/environment": "main",
    "lagoon.sh/service": "cli",
    "lagoon.sh/insightsType": ["sbom", "image"],
    "lagoon.sh/insightsOutputCompressed": ["true", "false" (default)] (optional),
    "lagoon.sh/insightsOutputFileExt": ["json (default)", "txt", "csv", "html", "jpg"] (optional),
    "lagoon.sh/insightsOutputFileMIMEType": ["text/html", "image/svg+xml"]  (optional)
  }
```



## Lagoon cli integration

```
lagoon insights [query]

lagoon insights builds --project "high-cotton"

| Project Name | Number of Builds | Failed Builds | Successful Builds | Builds per Month |
|--------------|-----------------|---------------|-------------------|------------------|
| high-cotton  |       100       |      20       |        80         |       25         |
|--------------|-----------------|---------------|-------------------|------------------|

lagoon insights builds --project "high-cotton" --range "-90d"

| Project Name | Number of Builds | Failed Builds | Successful Builds | Builds per Month | Builds in Last 90 Days |
|--------------|-----------------|---------------|-------------------|------------------|-----------------------|
| high-cotton  |       100       |      20       |        80         |       25         |          50           |
|--------------|-----------------|---------------|-------------------|------------------|-----------------------|

```

## Grafana / Prometheus interation



## Comms between insights handler REST and lagoon graphql api


1. Use direct api-to-api synchronous comms using HTTP calls


2. Asynchromous Event driven comms via NATs/Broker - lagoon api publishes events related to insights and the handler will subscribe and respond to those event payloads and update db accordingly.


3. Periodic data syncing between insights handler and Lagoon API via cron