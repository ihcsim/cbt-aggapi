FROM golang:1.17 as builder
WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
COPY vendor vendor
COPY pkg pkg
COPY ./cmd/apiserver/main.go main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o apiserver main.go

FROM gcr.io/distroless/static-debian11
WORKDIR /
COPY --from=builder /workspace/apiserver .
ENTRYPOINT ["/apiserver"]
