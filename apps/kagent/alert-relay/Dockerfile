FROM golang:1.26@sha256:2005724102f45917a63e9d092fc0e4ea56ea575048ce147caad5f5f61502c365 AS build

WORKDIR /src
COPY go.mod .
COPY src/ ./src/
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/alert-relay ./src

FROM gcr.io/distroless/static-debian12:nonroot@sha256:1b7b9f0f0e0a1d2155f531db587cc48ec26aaf97ab64364225f5bf18a054e66a

COPY --from=build /out/alert-relay /alert-relay

USER 1000:1000
EXPOSE 8080

ENTRYPOINT ["/alert-relay"]
