FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/throne-ingest ./cmd/throne-ingest

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/throne-ingest /throne-ingest
EXPOSE 8080 8443
USER nonroot:nonroot
ENTRYPOINT ["/throne-ingest"]
