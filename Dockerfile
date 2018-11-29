#FROM golang:1.11
FROM alexellis2/go-armhf:1.7.4

WORKDIR /src
COPY . /src

RUN go build -o butler -mod=vendor main.go
RUN mv butler /butler

EXPOSE 80
