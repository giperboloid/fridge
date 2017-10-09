FROM alpine
MAINTAINER Molchanov Kostiantyn (kostyamol@gmail.com)

RUN mkdir -p /home/fridgems/bin

WORKDIR /home/fridgems/bin
COPY ./cmd/fridgems .

RUN \  
    chown daemon fridgems && \
    chmod +x fridgems
    
USER daemon
ENTRYPOINT ["./fridgems"]
CMD ["LG", "FF:FF:FF:FF:FF:FF"]
