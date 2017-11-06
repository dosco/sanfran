FROM alpine:3.6

ADD build/sanfran-sidecar /
WORKDIR /

CMD [ "./sanfran-sidecar", "-logtostderr" ]

EXPOSE 8080
