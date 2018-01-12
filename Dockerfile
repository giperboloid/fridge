FROM alpine:latest
MAINTAINER Kostiantyn Molchanov (kostyamol@gmail.com)

RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY ./cmd/fridgems/fridgems .

RUN \  
    chown daemon fridgems && \
    chmod +x fridgems    
USER daemon

ENTRYPOINT ["./fridgems"]
CMD ["-name=LG", "-mac=FF:FF:FF:FF:FF:FF"]
