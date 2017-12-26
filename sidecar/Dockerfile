FROM mhart/alpine-node:base-9

ADD build/sanfran-sidecar /
WORKDIR /

VOLUME [ "/shared" ]

CMD [ "./sanfran-sidecar", "-logtostderr" ]

EXPOSE 8080
