FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/storesvc ./cmd/storesvc
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrator ./cmd/migrator

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/storesvc /storesvc
COPY --from=build /out/migrator /migrator
COPY --from=build /src/migrations /migrations
EXPOSE 8081
USER nonroot:nonroot
ENTRYPOINT ["/storesvc"]
