FROM ubuntu:jammy

ENV DEBIAN_FRONTEND=noninteractive

ARG cnb_uid=0
ARG cnb_gid=0

USER ${cnb_uid}:${cnb_gid}

COPY entrypoint /entrypoint
COPY fixtures /fixtures

ENTRYPOINT ["/entrypoint"]
