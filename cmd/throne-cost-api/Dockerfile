FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/throne-cost-api ./cmd/throne-cost-api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/throne-cost-api /throne-cost-api
USER nonroot:nonroot
ENTRYPOINT ["/throne-cost-api"]
