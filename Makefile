.POSIX:
.SUFFIXES:
.PHONY: debug release vet clean version push all tag upload install_vendor
.SILENT: version dockerbuild

SOURCES = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
GITHUB_TOKEN = $(shell cat token)

BINARY=sluggissh
FULL=github.com/stephane-martin/sluggissh
VERSION=0.1.0
LDFLAGS=-ldflags '-X main.Version=${VERSION}'
LDFLAGS_RELEASE=-ldflags '-w -s -X main.Version=${VERSION}'

debug: ${BINARY}-debug
release: ${BINARY}

install_vendor:
	go install -i ./vendor/...

tag:
	dep ensure
	git add .
	git commit -m "Version ${VERSION}"
	git tag -a ${VERSION} -m "Version ${VERSION}"
	git push
	git push --tags

upload: all
	github-release ${VERSION} sluggissh_linux_amd64 sluggissh_linux_arm sluggissh_linux_arm64 sluggissh_openbsd_amd64 sluggissh_freebsd_amd64 --tag ${VERSION} --github-repository stephane-martin/sluggissh --github-access-token ${GITHUB_TOKEN}

${BINARY}-debug: ${SOURCES} ${STATICFILES} 
	dep ensure
	CGO_ENABLED=0 go build -x -tags 'netgo osusergo' -o ${BINARY}-debug ${LDFLAGS} ${FULL}

${BINARY}: ${SOURCES} ${STATICFILES} model_gen.go .tools_sync
	dep ensure
	CGO_ENABLED=0 go build -a -installsuffix nocgo -tags 'netgo osusergo' -o ${BINARY} ${LDFLAGS_RELEASE} ${FULL}

${BINARY}_openbsd_amd64: ${SOURCES} ${STATICFILES} model_gen.go .tools_sync
	dep ensure
	GOOS=openbsd GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix nocgo -tags 'netgo osusergo' -o ${BINARY}_openbsd_amd64 ${LDFLAGS_RELEASE} ${FULL}

${BINARY}_freebsd_amd64: ${SOURCES} ${STATICFILES} model_gen.go .tools_sync
	dep ensure
	GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix nocgo -tags 'netgo osusergo' -o ${BINARY}_freebsd_amd64 ${LDFLAGS_RELEASE} ${FULL}

${BINARY}_linux_amd64: ${SOURCES} ${STATICFILES} model_gen.go .tools_sync
	dep ensure
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix nocgo -tags 'netgo osusergo' -o ${BINARY}_linux_amd64 ${LDFLAGS_RELEASE} ${FULL}

${BINARY}_linux_arm: ${SOURCES} ${STATICFILES} model_gen.go .tools_sync
	dep ensure
	GOOS=linux GOARCH=arm CGO_ENABLED=0 go build -a -installsuffix nocgo -tags 'netgo osusergo' -o ${BINARY}_linux_arm ${LDFLAGS_RELEASE} ${FULL}

${BINARY}_linux_arm64: ${SOURCES} ${STATICFILES} model_gen.go .tools_sync
	dep ensure
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix nocgo -tags 'netgo osusergo' -o ${BINARY}_linux_arm64 ${LDFLAGS_RELEASE} ${FULL}


all: ${BINARY}_openbsd_amd64 ${BINARY}_freebsd_amd64 ${BINARY}_linux_amd64 ${BINARY}_linux_arm ${BINARY}_linux_arm64 README.rst

clean:
	rm -f ${BINARY} ${BINARY}_debug

version:
	echo ${VERSION}

vet:
	go vet ./...


