FROM golang:1.27.3-alpine AS build

WORKDIR /src/apps/backend

COPY apps/backend/go.mod apps/backend/go.sum ./
RUN go mod download

COPY apps/backend/ ./
RUN go run ./cmd/template-registry
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot AS api

COPY --from=build --chown=nonroot:nonroot /out/api /api

EXPOSE 8080

ENTRYPOINT ["/api"]

FROM gcr.io/distroless/static-debian12:nonroot AS server

COPY --from=build --chown=nonroot:nonroot /out/server /server

ENTRYPOINT ["/server"]
