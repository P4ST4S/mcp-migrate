FROM golang:1.21-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /out/mcp-migrate ./cmd/mcp-migrate

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/mcp-migrate /mcp-migrate
ENTRYPOINT ["/mcp-migrate"]
