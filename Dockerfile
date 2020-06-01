# If you're not on Linux/AMD64, FROM golang:latest should work fine.
# Alpine image doesn't include /etc/mime.types
#FROM golang@sha256:c11dac79f0b25f3f3188429f8dd37cb6004021624b42085f745cb907ca1560a9
FROM golang@sha256:8b98da51cbc03732a136fd9ed4449683d5b6976debd9d89403070a3390c1b3d8

LABEL maintainer="Teran"

WORKDIR /app

COPY . .

## Run tests + build
RUN apt-get update
RUN apt-get install shellcheck
# Yes, this does both.
RUN ./test.sh
##
# Or, just build.
# RUN go build
##

EXPOSE 4444

VOLUME decensor

ENV DECENSOR_DIR /decensor

CMD ["./decensor", "web", ":4444"]
