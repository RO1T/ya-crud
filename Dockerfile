FROM golang:latest AS build

WORKDIR /app
ADD main.go .
RUN GOARCH=amd64 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o server-app *.go

FROM scratch
COPY --from=build /app/server-app /server-app

ENTRYPOINT ["/server-app"]
