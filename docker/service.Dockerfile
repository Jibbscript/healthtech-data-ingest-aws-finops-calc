FROM golang:1.23-alpine AS build
WORKDIR /src

ARG SERVICE
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN case "$SERVICE" in \
      throne-api|throne-cost-api|throne-inference|throne-ingest|throne-load|throne-processor) ;; \
      *) echo "unsupported SERVICE=$SERVICE" >&2; exit 1 ;; \
    esac && CGO_ENABLED=0 GOOS=linux go build -o /out/service "./cmd/${SERVICE}"

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/service /service
USER nonroot:nonroot
ENTRYPOINT ["/service"]
