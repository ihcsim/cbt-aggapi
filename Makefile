IMAGE_REPO_AGGAPI ?= quay.io/isim/cbt-aggapi
IMAGE_REPO_GRPC ?= quay.io/isim/cbt-grpc
IMAGE_TAG_AGGAPI ?= latest
IMAGE_TAG_GRPC ?= latest

init_repo:
	apiserver-boot init repo --domain storage.k8s.io

create_group:
	apiserver-boot create group version resource --group cbt --version v1alpha1 --kind VolumeSnapshotDelta

build:
	apiserver-boot build executables --targets apiserver

image: build
	apiserver-boot build container --targets apiserver --image $(IMAGE_REPO_AGGAPI):$(IMAGE_TAG_AGGAPI)
	docker build -t $(IMAGE_REPO_GRPC):$(IMAGE_TAG_GRPC) -f Dockerfile-grpc .

push:
	docker push $(IMAGE_REPO_AGGAPI):$(IMAGE_TAG_AGGAPI)
	docker push $(IMAGE_REPO_GRPC):$(IMAGE_TAG_GRPC)

run-local:
	PATH=`pwd`/bin:${PATH} apiserver-boot run local --run "etcd,apiserver"

.PHONY: codegen
codegen:
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: proto
proto:
	protoc -I=proto \
		--go_out=pkg/grpc --go_opt=paths=source_relative \
   	--go-grpc_out=pkg/grpc --go-grpc_opt=paths=source_relative \
		proto/cbt.proto

.PHONY: yaml
yaml:
	apiserver-boot build config --name cbt --namespace csi-cbt --image $(IMAGE_REPO_AGGAPI):$(IMAGE_TAG_AGGAPI) --output config

deploy:
	kubectl apply -f config
