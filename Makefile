IMAGE_REPO_AGGAPI ?= quay.io/isim/cbt-aggapi
IMAGE_REPO_GRPC ?= quay.io/isim/cbt-grpc
IMAGE_REPO_HTTP ?= quay.io/isim/cbt-http
IMAGE_TAG_AGGAPI ?= latest
IMAGE_TAG_GRPC ?= latest
IMAGE_TAG_HTTP ?= latest

API_GROUP ?= cbt
API_VERSION ?= v1alpha1
API_KIND ?= VolumeSnapshotDelta

GOOS ?= linux
GOARCH ?= amd64

init_repo:
	apiserver-boot init repo --domain storage.k8s.io

create_group:
	apiserver-boot create group version resource --group $(API_GROUP) --version $(API_VERSION) --kind $(API_KIND)

apiserver:
	apiserver-boot build executables --targets apiserver

mock:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -a -o grpc-server ./cmd/mock/grpc/main.go
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -a -o http-server ./cmd/mock/http/main.go

build: apiserver mock

image: build
	apiserver-boot build container --targets apiserver --image $(IMAGE_REPO_AGGAPI):$(IMAGE_TAG_AGGAPI)
	docker build -t $(IMAGE_REPO_GRPC):$(IMAGE_TAG_GRPC) -f Dockerfile-grpc .
	docker build -t $(IMAGE_REPO_HTTP):$(IMAGE_TAG_HTTP) -f Dockerfile-http .

push:
	docker push $(IMAGE_REPO_AGGAPI):$(IMAGE_TAG_AGGAPI)
	docker push $(IMAGE_REPO_GRPC):$(IMAGE_TAG_GRPC)
	docker push $(IMAGE_REPO_HTTP):$(IMAGE_TAG_HTTP)

run-local:
	PATH=`pwd`/bin:${PATH} apiserver-boot run local --run apiserver

codegen: proto
	./hack/update-codegen.sh

codegen-verify:
	./hack/verify-codegen.sh

.PHONY: proto
proto:
	protoc -I=proto \
		--go_out=pkg/grpc --go_opt=paths=source_relative \
   	--go-grpc_out=pkg/grpc --go-grpc_opt=paths=source_relative \
		proto/cbt.proto

.PHONY: yaml
yaml:
	rm -rf yaml-generated
	apiserver-boot build config --name cbt --namespace csi-cbt --image $(IMAGE_REPO_AGGAPI):$(IMAGE_TAG_AGGAPI) --output yaml-generated

etcd:
	kubectl apply -f yaml/etcd.yaml

deploy:
	kubectl apply -f yaml
