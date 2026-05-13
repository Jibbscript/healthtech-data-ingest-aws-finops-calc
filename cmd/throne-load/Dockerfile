FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/throne-load ./cmd/throne-load

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/throne-load /throne-load
USER nonroot:nonroot
ENTRYPOINT ["/throne-load"]
