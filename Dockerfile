FROM golang:1.11

WORKDIR /go/src/github.com/ninnemana/butler
COPY . .

RUN go install -v ./...

EXPOSE 80

CMD ["butler"]
