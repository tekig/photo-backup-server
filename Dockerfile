FROM golang:1.24 AS build

WORKDIR /app

COPY go.mod go.sum .

RUN go mod download

COPY . .

RUN go build -o /photo-backup /app/cmd/photo-backup/.

FROM debian:bookworm-slim

WORKDIR /app

RUN DEBIAN_FRONTEND=noninteractive apt update && apt install -y ffmpeg imagemagick libheif1 && rm -rf /var/lib/apt/lists/*

COPY --from=build /photo-backup .

CMD ["/app/photo-backup"]