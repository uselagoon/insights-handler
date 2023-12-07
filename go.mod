module github.com/uselagoon/lagoon/services/insights-handler

go 1.21

require (
	github.com/Khan/genqlient v0.6.0
	github.com/cheshir/go-mq v1.0.2
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/joho/godotenv v1.5.1
	github.com/matryer/try v0.0.0-20161228173917-9ac251b645a2
	github.com/stretchr/testify v1.8.4
)

require (
	github.com/NeowayLabs/wabbit v0.0.0-20200409220312-12e68ab5b0c6 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.5 // indirect
	github.com/klauspost/cpuid v1.3.1 // indirect
	github.com/klauspost/cpuid/v2 v2.0.4 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/minio/md5-simd v1.1.0 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/moby/sys/mount v0.3.3 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/rs/xid v1.2.1 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/streadway/amqp v0.0.0-20200108173154-1c71cc93ed71 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/vektah/gqlparser/v2 v2.5.10 // indirect
	golang.org/x/crypto v0.16.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)

replace (
	github.com/docker/cli => github.com/docker/cli v20.10.19+incompatible
	github.com/docker/docker => github.com/docker/docker v20.10.19+incompatible
)

require (
	github.com/CycloneDX/cyclonedx-go v0.7.2-0.20230625092137-07e2f29defc3
	github.com/cheekybits/is v0.0.0-20150225183255-68e9c0620927 // indirect
	github.com/fsouza/go-dockerclient v1.7.3 // indirect
	github.com/minio/minio-go/v7 v7.0.21
	github.com/tiago4orion/conjure v0.0.0-20150908101743-93cb30b9d218 // indirect
	golang.org/x/net v0.19.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
)

// Fixes for AppID
replace github.com/cheshir/go-mq v1.0.2 => github.com/shreddedbacon/go-mq v0.0.0-20200419104937-b8e9af912ead

replace github.com/NeowayLabs/wabbit v0.0.0-20200409220312-12e68ab5b0c6 => github.com/shreddedbacon/wabbit v0.0.0-20200419104837-5b7b769d7204
