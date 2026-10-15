# Build stage: Go-ийн бүрэн image дээр binary бүтээнэ
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
# CGO унтраавал статик binary гарч, жижиг image-д ажиллана
RUN CGO_ENABLED=0 go build -o /raft-kv ./cmd/raft-kv

# Runtime stage: shell-тэй жижиг image (pod-ийн ordinal-аас ID авахад хэрэгтэй)
FROM alpine:3.20
WORKDIR /app
COPY --from=build /raft-kv /app/raft-kv
# data/node-N.bin энд хадгалагдана (/app/data)
ENTRYPOINT ["/app/raft-kv"]