FROM ubuntu:22.04

RUN apt-get -y update
RUN apt-get -y install build-essential curl zlib1g zlib1g-dev libssl-dev libpcre3 libpcre3-dev

COPY entrypoint /entrypoint

ENTRYPOINT ["/entrypoint"]
