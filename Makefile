.PHONY: gettrivy
gettrivy:
	mkdir -p internal/handler/testassets/bin/trivy/ && wget -O - https://github.com/aquasecurity/trivy/releases/download/v0.45.0/trivy_0.45.0_Linux-64bit.tar.gz | tar -zxvf - -C internal/handler/testassets/bin/trivy/


