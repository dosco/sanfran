FROM mhart/alpine-node:base-9

ENV TINI_VERSION v0.16.1-sf
ADD https://github.com/dosco/tini/releases/download/${TINI_VERSION}/tini-static /tini
RUN chmod +x /tini
ENTRYPOINT ["/tini", "-r", "--"]

ARG NODE_ENV
ENV NODE_ENV $NODE_ENV

RUN mkdir -p /app/node_modules
WORKDIR /app

COPY node_modules /app/node_modules/
COPY server.js /app/

VOLUME [ "/shared" ]

CMD ["/usr/bin/node", "server.js", "--max-old-space-size=128"]

EXPOSE 8081