FROM golang@sha256:c11dac79f0b25f3f3188429f8dd37cb6004021624b42085f745cb907ca1560a9

LABEL maintainer="Teran"

WORKDIR /app

COPY . .

RUN go build

EXPOSE 4444

VOLUME decensor

ENV DECENSOR_DIR /decensor

CMD ["./decensor", "web", ":4444"]
