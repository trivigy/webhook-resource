FROM golang:alpine

ARG pkg=webhook

RUN apk add --no-cache ca-certificates

COPY . $GOPATH/src/$pkg

RUN set -ex \
      && apk add --no-cache --virtual .build-deps \
              git \
      && go get -v $pkg/... \
      && apk del .build-deps

RUN go install $pkg/...

WORKDIR $GOPATH/src/$pkg
RUN mkdir -p /opt/resource && \
    go build -o /opt/resource/check ./check && \
    go build -o /opt/resource/in ./in && \
    go build -o /opt/resource/out ./out

WORKDIR $GOPATH
CMD echo "Use the webhook commands."; exit 1