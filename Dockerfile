FROM golang:1.24 AS build

WORKDIR /app

COPY go.mod go.sum .

RUN go mod download

COPY . .

RUN go build -o /photo-backup /app/cmd/photo-backup/.

FROM ubuntu:25.10

WORKDIR /app

RUN DEBIAN_FRONTEND=noninteractive apt update && DEBIAN_FRONTEND=noninteractive apt install -y ffmpeg imagemagick libheif1 && rm -rf /var/lib/apt/lists/*

COPY --from=build /photo-backup .

CMD ["/app/photo-backup"]