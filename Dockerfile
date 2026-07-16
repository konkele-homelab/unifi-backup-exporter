FROM golang:1.24-alpine AS build

WORKDIR /src

COPY go.mod .
RUN go mod download

COPY unifi-backup-exporter ./unifi-backup-exporter

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /unifi-backup-exporter \
    ./unifi-backup-exporter

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /unifi-backup-exporter /unifi-backup-exporter

EXPOSE 8081

ENTRYPOINT ["/unifi-backup-exporter"]