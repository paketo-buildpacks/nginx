FROM paketobuildpacks/build-jammy-base:0.1.14

ARG cnb_uid=0
ARG cnb_gid=0

USER ${cnb_uid}:${cnb_gid}

COPY entrypoint /entrypoint
COPY fixtures /fixtures

ENTRYPOINT ["/entrypoint"]
