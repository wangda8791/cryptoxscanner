FROM centos:7

RUN yum -y install epel-release && \
    yum -y install \
    	git \
	make \
	gcc \
	gcc-c++ \
	mingw64-gcc \
	zip \
	clang \
	which \
	patch \
	curl \
        sudo

ENV NODE_VERSION 10.15.0
RUN cd /usr/local && \
	curl -L -o - https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.gz | tar zxf - --strip-components=1


ENV GO_VERSION 1.12.6
RUN cd /usr/local && \
	curl -L -o - https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz | tar zxf -

RUN useradd --password "" --groups wheel builder
RUN echo "builder ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/builder

ENV PATH=/usr/local/go/bin:$PATH

COPY /build/docker-entrypoint.sh /
ENTRYPOINT ["/docker-entrypoint.sh"]
