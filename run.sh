#!/bin/bash

go run main.go \
    -rabbitmq-username guest \
    -rabbitmq-password guest \
    -lagoon-api-host http://localhost:3000/graphql \
    --jwt-token-signing-key super-secret-string  \
    --access-key-id minio \
    --secret-access-key minio123
