FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/throne-processor ./cmd/throne-processor

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/throne-processor /throne-processor
USER nonroot:nonroot
ENTRYPOINT ["/throne-processor"]
