FROM golang:1.27@sha256:2920bb124f1a2e12a1d39f0cab756ae0068441ee1da7655dec7f5e374944095b AS build

WORKDIR /src
COPY go.mod .
COPY src/ ./src/
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/alert-relay ./src

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

COPY --from=build /out/alert-relay /alert-relay

USER 1000:1000
EXPOSE 8080

ENTRYPOINT ["/alert-relay"]
