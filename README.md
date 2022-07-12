## CBT Controller

This repository contains an [aggregated API server] prototype used to serve the
[CSI changed block tracking] API.

The primary goal is to explore ways to implement an in-cluster API endpoint
which can be used to retrieve a long list of changed block entries (in the order
of hundreds of MiB), without putting the Kubernetes API server and etcd in the
data retrieval path.

This prototype explores:

* The implementation of a non-etcd [`custom` storage] that supports the
Kubernetes [`rest.Connecter`] interface
* The implementation of a [custom `Option`] that converts `url.Values` to a
`runtime.Object`

The `rest.Connecter` implements a modified `GET` handler to return a collection
of changed block entries, if the `fetchcbd` query parameter is defined.

Essentially, the default API endpoint will return a `VolumeSnapshotDelta` custom
resource:

```sh
$ curl --silent -k "https://localhost:9443/apis/cbt.storage.k8s.io/v1alpha1/volumesnapshotdelta/foo" | jq .
{
  "metadata": {
    "name": "test-delta",
    "namespace": "default",
    "creationTimestamp": null
  },
  "spec": {
    "baseVolumeSnapshotName": "base",
    "targetVolumeSnapshotName": "target"
  },
  "status": {
    "callbackURL": "example.com"
  }
}
```

Appending the API endpoint with the `fetchcbd=true` query parameter will return
the list of changed block entries:

```sh
$ curl --silent -k "https://localhost:9443/apis/cbt.storage.k8s.io/v1alpha1/volumesnapshotdelta/foo?fetchcbd=true" | jq .
[
  {
    "offset": 0,
    "blockSizeBytes": 524288,
    "dataToken": {
      "token": "ieEEQ9Bj7E6XR",
      "issuanceTime": "2022-07-11T22:06:20Z",
      "ttl": "3h0m0s"
    }
  },
  {
    "offset": 1,
    "blockSizeBytes": 524288,
    "dataToken": {
      "token": "widvSdPYZCyLB",
      "issuanceTime": "2022-07-11T22:06:20Z",
      "ttl": "3h0m0s"
    }
  },
  {
    "offset": 2,
    "blockSizeBytes": 524288,
    "dataToken": {
      "token": "VtSebH83xYzvB",
      "issuanceTime": "2022-07-11T22:06:20Z",
      "ttl": "3h0m0s"
    }
  }
]
```

Most of the setup code of the aggregated API server is generated using the
[`apiserver-builder`] tool.

## Quick Start

Install the `apiserver-builder` tool following the instructions
[here](https://github.com/kubernetes-sigs/apiserver-builder-alpha#installation).
The `apiserver-boot` tool requires the code to be checked out into the local
`$GOPATH` i.e. `github.com/ihcsim/cbt-controller`.

### Development

To run the tests:

```sh
go test ./...
```

To re-generate the Go code:

```sh
make generate CONTROLLER_GEN=$GOPATH/bin/controller-gen
```

To build the binaries:

```sh
apiserver-boot build executables
```

To start the mock GRPC server that serves the sample CBT records:

```sh
go run ./cmd/server
```

To run the aggregated API server locally:

```sh
PATH=`pwd`/bin:$PATH apiserver-boot run local --run "etcd,apiserver"
```

### Working With Custom Resource

```sh
cat<<EOF | kubectl apply -f -
apiVersion: cbt.storage.k8s.io/v1alpha1
kind: VolumeSnapshotDelta
metadata:
  name: test-delta
  namespace: default
spec:
  baseVolumeSnapshotName: vs-00
  targetVolumeSnapshotName: vs-01
  mode: block
EOF
```

### Testing On Kubernetes

Define the repository URL and tag for your image:

```sh
export IMAGE_REPO=<your_image_repo>

export IMAGE_TAG=<your_image_tag>
```

Output the Kubernetes YAML manifest to a `config` folder:

```sh
apiserver-boot build config
  --name cbt \
  --namespace csi-cbt \
  --image ${IMAGE_REPO}:${IMAGE_TAG} \
  --output config
```

Build and push the aggregated API server image:

```sh
apiserver-boot build container \
  --targets apiserver \
  --image ${IMAGE_REPO}:${IMAGE_TAG}

docker push ${IMAGE_REPO}:${IMAGE_TAG}
```

## License

Apache License 2.0, see [LICENSE].

[aggregated API server ]:https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/
[CSI changed block tracking]: https://github.com/kubernetes/enhancements/pull/3367
[`rest.Connecter`]: https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Connecter
[`custom` storage]: pkg/storage/custom.go
[custom `Option`]: pkg/apis/cbt/v1alpha1/volumesnapshotdeltaoption_types.go
[`apiserver-builder`]: https://github.com/kubernetes-sigs/apiserver-builder-alpha
[LICENSE]: LICENSE
