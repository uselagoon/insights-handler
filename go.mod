module github.com/uselagoon/lagoon/services/insights-handler

go 1.21

require (
	github.com/Khan/genqlient v0.7.0
	github.com/cheshir/go-mq v1.0.2
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gin-gonic/gin v1.9.1
	github.com/joho/godotenv v1.5.1
	github.com/matryer/try v0.0.0-20161228173917-9ac251b645a2
	github.com/stretchr/testify v1.9.0
	gorm.io/driver/postgres v1.5.9
	gorm.io/driver/sqlite v1.5.4
	gorm.io/gorm v1.25.10
)

require (
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/NeowayLabs/wabbit v0.0.0-20200409220312-12e68ab5b0c6 // indirect
	github.com/bytedance/sonic v1.9.1 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/containerd/containerd v1.7.13 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.14.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.5.5 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.5 // indirect
	github.com/klauspost/cpuid v1.3.1 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mattn/go-sqlite3 v1.14.18 // indirect
	github.com/minio/md5-simd v1.1.0 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc2.0.20221005185240-3a7f492d3f1b // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/rs/xid v1.2.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/streadway/amqp v0.0.0-20200108173154-1c71cc93ed71 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	github.com/vektah/gqlparser/v2 v2.5.11 // indirect
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/crypto v0.19.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)

replace (
	github.com/docker/cli => github.com/docker/cli v25.0.3+incompatible
	github.com/docker/docker => github.com/docker/docker v25.0.3+incompatible
)

require (
	github.com/CycloneDX/cyclonedx-go v0.8.0
	github.com/cheekybits/is v0.0.0-20150225183255-68e9c0620927 // indirect
	github.com/fsouza/go-dockerclient v1.7.3 // indirect
	github.com/minio/minio-go/v7 v7.0.21
	github.com/tiago4orion/conjure v0.0.0-20150908101743-93cb30b9d218 // indirect
	golang.org/x/net v0.21.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
)

// Fixes for AppID
replace github.com/cheshir/go-mq v1.0.2 => github.com/shreddedbacon/go-mq v0.0.0-20200419104937-b8e9af912ead

replace github.com/NeowayLabs/wabbit v0.0.0-20200409220312-12e68ab5b0c6 => github.com/shreddedbacon/wabbit v0.0.0-20200419104837-5b7b769d7204
