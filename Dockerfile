FROM golang:1.11

WORKDIR /src
COPY . /src

RUN go build -o butler -mod=vendor main.go
RUN mv butler /butler

EXPOSE 80

ENTRYPOINT ["/butler"]
