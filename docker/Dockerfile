# Reproducible Buildroot host environment (Ubuntu 24.04 ≈ GitHub Actions runner).
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

COPY docker/install-buildroot-deps.sh docker/install-go.sh /usr/local/bin/
COPY go.mod /usr/local/share/bigfred-os-go.mod
ENV GO_MOD=/usr/local/share/bigfred-os-go.mod
RUN chmod +x /usr/local/bin/install-buildroot-deps.sh /usr/local/bin/install-go.sh \
	&& /usr/local/bin/install-buildroot-deps.sh \
	&& /usr/local/bin/install-go.sh \
	&& rm -rf /var/lib/apt/lists/* \
	&& mkdir -p /work \
	&& if getent passwd 1000 >/dev/null; then \
		usermod -d /work -s /bin/bash "$(getent passwd 1000 | cut -d: -f1)"; \
	   else \
		useradd -u 1000 -g "$(getent group 1000 | cut -d: -f1)" -d /work -M -s /bin/bash builder; \
	   fi \
	&& chown 1000:1000 /work

WORKDIR /work
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOTOOLCHAIN=local
USER 1000:1000
