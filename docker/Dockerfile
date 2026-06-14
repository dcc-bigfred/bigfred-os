# Reproducible Buildroot host environment (Ubuntu 24.04 ≈ GitHub Actions runner).
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

COPY install-buildroot-deps.sh /usr/local/bin/install-buildroot-deps.sh
RUN chmod +x /usr/local/bin/install-buildroot-deps.sh \
	&& /usr/local/bin/install-buildroot-deps.sh \
	&& apt-get install -y --no-install-recommends golang-go \
	&& rm -rf /var/lib/apt/lists/* \
	&& mkdir -p /work \
	&& if getent passwd 1000 >/dev/null; then \
		usermod -d /work -s /bin/bash "$(getent passwd 1000 | cut -d: -f1)"; \
	   else \
		useradd -u 1000 -g "$(getent group 1000 | cut -d: -f1)" -d /work -M -s /bin/bash builder; \
	   fi \
	&& chown 1000:1000 /work

WORKDIR /work
USER 1000:1000
