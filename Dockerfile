FROM golang:alpine as builder
LABEL maintainer "Artur Sak <artursak1994@gmail.com>"

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN	apk add --no-cache \
	bash \
	ca-certificates

COPY . /go/src/github.com/projects/gitmv

RUN set -x \
	&& apk add --no-cache --virtual .build-deps \
		git \
		gcc \
		libc-dev \
		libgcc \
		make \
	&& cd /go/src/github.com/projects/gitmv \
	&& make static \
	&& mv gitmv /usr/bin/gitmv \
	&& apk del .build-deps \
	&& rm -rf /go \
	&& echo "Build complete."

FROM alpine:latest

COPY --from=builder /usr/bin/gitmv /usr/bin/gitmv
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs

ENTRYPOINT [ "gitmv" ]
CMD [ "--help" ]
