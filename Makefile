.PHONY: gettrivy
gettrivy:
	mkdir -p internal/handler/testassets/bin/trivy/ && wget -O - https://github.com/aquasecurity/trivy/releases/download/v0.45.0/trivy_0.45.0_Linux-64bit.tar.gz | tar -zxvf - -C internal/handler/testassets/bin/trivy/


.PHONY: runlocal
runlocal:
	go run main.go --problems-from-sbom=true --rabbitmq-username=guest  --rabbitmq-password=guest --lagoon-api-host=http://localhost:8888/graphql   --jwt-token-signing-key=secret   --access-key-id=minio     --secret-access-key=minio123 --disable-s3-upload=true --debug=true


