FROM golang:1.23@sha256:60deed95d3888cc5e4d9ff8a10c54e5edc008c6ae3fba6187be6fb592e19e8c0 AS build

WORKDIR /src
COPY go.mod .
COPY src/ ./src/
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/alert-relay ./src

FROM gcr.io/distroless/static-debian12:nonroot@sha256:f5b485ea962d9bd1186b2f6b3a061191539b905b82ec395de78cbfae51f20e35

COPY --from=build /out/alert-relay /alert-relay

USER 1000:1000
EXPOSE 8080

ENTRYPOINT ["/alert-relay"]
