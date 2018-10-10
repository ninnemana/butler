# Butler

A simple golang reverse proxy that supports HTTP(S) and creates route targets
from a JSON configuration file.

## Building

To run the application locally:

```
> go run main.go --config=examples/config.json
> sudo sed -i "127.0.0.1 butler-proxy" /etc/hosts
> curl -H 'Host: butler-proxy' http://localhost:8080
> view the reddits...
```

To run the application as a docker image:

```
> docker build -t butler .
> docker run -p 8080:80 butler --config=examples/config.json
> sudo sed -i "127.0.0.1 butler-proxy" /etc/hosts
> curl -H 'Host: butler-proxy' http://localhost:8080
> view the reddits...
```

## Testing

```
> go test -v ./...
```
