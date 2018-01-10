FROM alpine:latest
MAINTAINER Kostiantyn Molchanov (kostyamol@gmail.com)

RUN apk --no-cache add ca-certificates

RUN mkdir -p /home/fridgems/bin

WORKDIR /home/fridgems/bin
COPY ./cmd/fridgems .

RUN \  
    chown daemon fridgems && \
    chmod +x fridgems
    
USER daemon
ENTRYPOINT ["./fridgems"]
CMD ["-name=LG", "-mac=FF:FF:FF:FF:FF:FF"]
