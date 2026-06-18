FROM golang:1.26 AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o wgmp .

FROM debian:trixie-slim
WORKDIR /app
# ENV DEBIAN_FRONTEND=noninteractive
# RUN apt update && \
#     apt install -y \
#     ca-certificates
# RUN rm -rf /var/lib/apt/lists/*

COPY --from=build /app/wgmp /app/wgmp
ENTRYPOINT ["./wgmp"]
CMD ["list-interfaces"]
