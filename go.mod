module github.com/uselagoon/lagoon/services/insights-handler

go 1.23.0

toolchain go1.24.6

require (
	github.com/Khan/genqlient v0.8.1
	github.com/cheshir/go-mq v1.2.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/joho/godotenv v1.5.1
	github.com/matryer/try v0.0.0-20161228173917-9ac251b645a2
	github.com/stretchr/testify v1.11.1
	github.com/uselagoon/machinery v0.0.34
)

require (
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/NeowayLabs/wabbit v0.0.0-20210927194032-73ad61d1620e // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.11 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/minio/crc64nvme v1.0.2 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/sys/user v0.1.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc2.0.20221005185240-3a7f492d3f1b // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/streadway/amqp v1.1.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/tinylib/msgp v1.3.0 // indirect
	github.com/vektah/gqlparser/v2 v2.5.21 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace (
	github.com/docker/cli => github.com/docker/cli v28.3.3+incompatible
	github.com/docker/docker => github.com/docker/docker v28.0.0+incompatible
)

require (
	github.com/CycloneDX/cyclonedx-go v0.9.2
	github.com/cheekybits/is v0.0.0-20150225183255-68e9c0620927 // indirect
	github.com/fsouza/go-dockerclient v1.7.3 // indirect
	github.com/minio/minio-go/v7 v7.0.95
	github.com/tiago4orion/conjure v0.0.0-20150908101743-93cb30b9d218 // indirect
	golang.org/x/net v0.41.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
)

// Fixes for AppID
replace github.com/cheshir/go-mq v1.0.2 => github.com/shreddedbacon/go-mq v0.0.0-20200419104937-b8e9af912ead

replace github.com/NeowayLabs/wabbit v0.0.0-20200409220312-12e68ab5b0c6 => github.com/shreddedbacon/wabbit v0.0.0-20200419104837-5b7b769d7204
