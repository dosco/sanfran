FROM alpine:3.6

ADD build/sanfran-api-proxy /
ADD fnapi.swagger.json /
WORKDIR /


CMD [ "./sanfran-api-proxy", "-logtostderr" ]

EXPOSE 8080
