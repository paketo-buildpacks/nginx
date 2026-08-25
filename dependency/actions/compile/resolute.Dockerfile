FROM ubuntu:resolute

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get -y update
RUN apt-get -y install build-essential curl zlib1g zlib1g-dev libssl-dev libpcre2-dev

COPY entrypoint /entrypoint

ENTRYPOINT ["/entrypoint"]