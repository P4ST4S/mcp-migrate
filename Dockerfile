FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /out/mcp-migrate ./cmd/mcp-migrate

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/mcp-migrate /mcp-migrate
ENTRYPOINT ["/mcp-migrate"]
