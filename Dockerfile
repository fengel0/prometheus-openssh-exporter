
FROM golang:1.22.5-alpine AS build

# Set the working directory inside the container
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build


FROM debian:stable-slim

RUN apt-get update -y
RUN apt-get upgrade -y
RUN apt-get install systemd -y
RUN apt-get install systemd -y

WORKDIR /app
COPY --from=build /app/prometheus-openssh-exporter .
RUN chmod +x ./prometheus-openssh-exporter

CMD [ "./prometheus-openssh-exporter" ]
