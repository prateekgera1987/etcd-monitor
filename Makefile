.PHONY:

GO_SOURCES=$(shell find . -name \*.go)
SOURCES=$(GO_SOURCES)
PLATFORM_BINARIES=dist/etcd-monitor-linux-amd64

IMAGE_NAME=kasko/etcd-monitor
GITHUB_USER=kasko
GITHUB_REPOSITORY=etcd-monitor

all: $(PLATFORM_BINARIES)

tools:
	go get -u github.com/aws/aws-sdk-go

clean:
	-rm $(PLATFORM_BINARIES)

dist/cacert.pem:
	[ -d dist ] || mkdir dist
	curl -s -o $@ https://curl.haxx.se/ca/cacert.pem

dist/etcd-monitor-linux-amd64: $(SOURCES)
	[ -d dist ] || mkdir dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags '-s' \
	  -o $@ ./etcd-monitor.go

container: dist/cacert.pem dist/etcd-monitor-linux-amd64
	docker build -t $(IMAGE_NAME) .

check:
	go test ./...

lint:
	go fmt ./...
	goimports -w $(GO_SOURCES)

release: lint check container $(PLATFORM_BINARIES)
	@[ ! -z "$(VERSION)" ] || (echo "you must specify the VERSION"; false)
	# which ghr >/dev/null || go get github.com/tcnksm/ghr
	# ghr -u $(GITHUB_USER) -r $(GITHUB_REPOSITORY) --delete v$(VERSION) dist/
	# docker tag $(IMAGE_NAME) $(IMAGE_NAME):$(VERSION)
	docker push $(IMAGE_NAME)
	# docker push $(IMAGE_NAME):$(VERSION)
