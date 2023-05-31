.PHONY: gettestgrype
gettestgrype:
	curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b ./internal/handler/testassets/bin