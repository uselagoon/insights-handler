# Lagoon Insights Handler

The Lagoon Insights Handler service is a primary hub for processing incoming data about Lagoon environments. 
These `insights` are gathered in remote clusters via the [insights-remote](https://github.com/uselagoon/insights-remote) service,
sent to the present service via the [Lagoon broker](https://github.com/uselagoon/lagoon/tree/main/services/broker) (presently a RabbitMQ instance), and then processed.


## Data types processed by the Insights Handler

Presently there are several data sources processed by the Insights Handler.

1. Image inspect information
   2. This is, essentially, the output of a `docker inspect` run at a project's build deploy time.
4. SBOM processing
   5. The [Lagoon Build Deploy Tool](https://github.com/uselagoon/build-deploy-tool) will run trivy on the resulting images that make up a project.
   6. This produces a Software Bill of Materials, essentially a list of all packages installed on those images.
   7. These SBOMs are processed and, optionally, run through Trivy to produce a list of vulnerabilities, which are written to Lagoon as `Problems`
8. Direct `Fact` and `Problem` writes to Lagoon
   9. Effectively the core part of these write described in the section of the Insights Remote documentation about [Insights written directly to insights remote](https://github.com/uselagoon/insights-remote?tab=readme-ov-file#insights-written-directly-to-insights-remote).

## Parsing, filtering, and transforming incoming data

The Insights Handler allows one to set up processing of incoming data to manipulate it before it is sent to Lagoon as Facts.
This is achieved by specifying so-called "filter-transformers", which are defined as YAML and passed to the handler on startup.

The assumption is that data coming from writes to the `insights remote` service endpoints will be formatted as one would like them to appear in the Lagoon API.
However, data from image inspect and SBOM generation processes are not under our control at generation time, therefore being able to manipulate them is useful.

Let's take a look at the structure of a simple filter-transformer

```
    - type: Package
      lookupvalue:
          - name: Name
            value: alpine
            exactMatch: true
      transformations:
          - name: Name
            value: Alpine Linux
          - name: Category
            value: OS
          - name: Description
            value: Base image Alpine Linux version
```

The `type` describes the source that this transformer is concerned with - this can be one of three values:
- Package - targets SBOMs
- EnvironmentVariable - will target env vars in image inspect data (i.e. if an `ENV` is part of the resultant image)
- InspectLabel - targets labels listed in image inspect data

The `lookupvalue` describes rules for matching the item in the incoming data. In the case above, we're looking for a Package `Name` that matches `alpine` exactly.

The `transformations` section describes what we'd like the resulting fact's data to be - so instead of `Name` being `alpine` (which is what we see in our target rule), we replace `alpine` with the text `Alpine Linux`.

There are several other examples in `default_filter_transformers.yaml`.

### Specifying alternative filters and transformations

The Insights Handler will load the default handlers found in the root directory `default_filter_transformers.yaml`.

This can be overridden in serveral ways - either by replacing the file itself (in, for instance, a configMap, etc.), via the flag `--filter-transformer-file`, or the env var `FILTER_TRANSFORMER_FILE`.

## Facts
Currently, the main purpose is to consume a Software Bill of Materials (SBOM) of facts from the logs queue, process
and push to the api and s3 bucket.


## Local development

Assuming that you're running a Lagoon development instance (i.e. pulling the Lagoon source and running `make up`), some reasonable defaults for developing locally could be given by the following: 

    go run main.go \
        -rabbitmq-username guest \
        -rabbitmq-password guest \
        -lagoon-api-host http://localhost:8888/graphql \
        --jwt-token-signing-key secret  \
        --access-key-id minio \
        --secret-access-key minio123 \
        --debug=true

To compile GraphQL schema, type-safe structs and response data with genqlient we just add a query/mutation inside of `lagoonclient/genqlient.graphql` and run:

    go generate
