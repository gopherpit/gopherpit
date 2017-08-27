FROM debian:stretch

MAINTAINER Janos Guljas <janos@resenje.org>

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY gopherpit /app/gopherpit
COPY version /app/version
COPY assets /app/assets
COPY static /app/static
COPY templates /app/templates
COPY docker/defaults /app/defaults

RUN ln -s /app/gopherpit /usr/local/bin/gopherpit

VOLUME /log /config /storage

EXPOSE 80 443 6060

ENV GOPHERPIT_CONFIGDIR=/config

ENTRYPOINT ["/app/gopherpit"]
