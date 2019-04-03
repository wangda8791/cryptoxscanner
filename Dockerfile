FROM centos:7

RUN yum -y install \
    	git \
	make \
	gcc \
	gcc-c++ \
	curl

ENV NODE_V=10.14.1	
RUN cd /usr/local && \
    curl --silent -L -o - https://nodejs.org/dist/v${NODE_V}/node-v${NODE_V}-linux-x64.tar.gz | tar zxf - --strip-components=1

ENV GO_V=1.11.2
RUN cd /usr/local && \
    curl --silent -L -o - https://dl.google.com/go/go${GO_V}.linux-amd64.tar.gz | tar zxf -
ENV PATH=/usr/local/go/bin:$PATH

WORKDIR /src
COPY / .
RUN make install-deps
RUN make

FROM centos:7
WORKDIR /app
COPY --from=0 /src/go/cryptoxscanner .
WORKDIR /data

VOLUME /data

CMD ["/app/cryptoxscanner", "server"]
