FROM golang:1.26 AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o wgmp .

FROM scratch
WORKDIR /app

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /app/wgmp /app/wgmp

USER 65532

ENTRYPOINT ["./wgmp"]
CMD ["list-interfaces"]
