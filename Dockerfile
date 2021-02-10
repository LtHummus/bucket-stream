FROM golang:1.15.7-alpine3.13

ENV GOOS=linux
WORKDIR /go/src/app
COPY . .

RUN go build -o /tmp/streamer cmd/main.go

FROM alpine:3.13

COPY --from=0 /tmp/streamer streamer

RUN /bin/sh -c "chmod +x streamer; apk add ffmpeg"

CMD "./streamer"
