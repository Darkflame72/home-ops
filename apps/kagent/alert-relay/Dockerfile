FROM golang:1.27@sha256:65b6f280bf050ec5af12716857e8ea8439d694dbba8f31ceeb7630670071f2bb AS build

WORKDIR /src
COPY go.mod .
COPY src/ ./src/
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/alert-relay ./src

FROM gcr.io/distroless/static-debian12:nonroot@sha256:1b7b9f0f0e0a1d2155f531db587cc48ec26aaf97ab64364225f5bf18a054e66a

COPY --from=build /out/alert-relay /alert-relay

USER 1000:1000
EXPOSE 8080

ENTRYPOINT ["/alert-relay"]
