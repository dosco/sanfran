FROM alpine:3.6

ADD build/sanfran-sidecar /
WORKDIR /

VOLUME [ "/shared" ]

CMD [ "./sanfran-sidecar", "-logtostderr" ]

EXPOSE 8080
