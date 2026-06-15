# syntax=docker/dockerfile:1
#
# Multi-stage Dockerfile that builds distribution packages for gwsm.
#
# Build a .deb:
#   docker build --target packager --build-arg VERSION=1.0.0 -f Dockerfile -t gwsm-deb .
#   docker run --rm -v $(pwd)/dist:/output gwsm-deb
#
# Build a .tar.gz:
#   docker build --target archiver --build-arg VERSION=1.0.0 -f Dockerfile -t gwsm-tar .
#   docker run --rm -v $(pwd)/dist:/output gwsm-tar
#
# Output lands in ./dist/

# ---------------------------------------------------------------------------
# Stage 1 — Build the Go binary
# CGO_ENABLED=0 produces a fully static binary with no shared library deps.
# ---------------------------------------------------------------------------
FROM golang:1.26.4-bookworm AS builder

WORKDIR /src

# Cache module downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w" -o gwsm ./cmd/gwsm

# ---------------------------------------------------------------------------
# Stage 2 — Assemble and build the .deb package
# ---------------------------------------------------------------------------
FROM debian:bookworm-slim AS packager

ARG VERSION=dev

RUN apt-get update \
    && apt-get install -y --no-install-recommends dpkg-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /pkg

# Detect host architecture using dpkg so the package metadata is correct.
RUN dpkg --print-architecture > /arch

# Build the directory tree that dpkg-deb will turn into a .deb.
RUN mkdir -p \
    deb/DEBIAN \
    deb/usr/bin \
    deb/usr/lib/systemd/user \
    deb/usr/share/doc/gwsm

# Binary
COPY --from=builder /src/gwsm deb/usr/bin/gwsm
RUN chmod 755 deb/usr/bin/gwsm

# Systemd user service — substitute placeholder with /usr/bin/gwsm for .deb installs
COPY packaging/gwsm.service deb/usr/lib/systemd/user/gwsm.service
RUN sed -i 's|{{GWSM_BIN}}|/usr/bin/gwsm|g' deb/usr/lib/systemd/user/gwsm.service

# Documentation
COPY gwsm.example.toml deb/usr/share/doc/gwsm/
COPY README.md         deb/usr/share/doc/gwsm/

# DEBIAN control files
COPY packaging/DEBIAN/control  deb/DEBIAN/control
COPY packaging/DEBIAN/postinst deb/DEBIAN/postinst
COPY packaging/DEBIAN/prerm    deb/DEBIAN/prerm
RUN chmod 755 deb/DEBIAN/postinst deb/DEBIAN/prerm

# Substitute {{VERSION}} and {{ARCH}} placeholders in the control file.
RUN ARCH=$(cat /arch) && \
    sed -i "s/{{VERSION}}/${VERSION}/g" deb/DEBIAN/control && \
    sed -i "s/{{ARCH}}/${ARCH}/g"       deb/DEBIAN/control

# Build the package.
RUN ARCH=$(cat /arch) && \
    dpkg-deb --build --root-owner-group deb \
        "gwsm_${VERSION}_${ARCH}.deb"

CMD ["sh", "-c", "cp /pkg/gwsm_*.deb /output/"]

# ---------------------------------------------------------------------------
# Stage 3 — Assemble and build the .tar.gz archive
# ---------------------------------------------------------------------------
FROM debian:bookworm-slim AS archiver

ARG VERSION=dev

RUN apt-get update \
    && apt-get install -y --no-install-recommends dpkg-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /pkg

RUN dpkg --print-architecture > /arch

# Build the tarball directory structure.
RUN ARCH=$(cat /arch) && mkdir -p "gwsm-${VERSION}-linux-${ARCH}"

COPY --from=builder /src/gwsm         .
COPY packaging/gwsm.service           .
COPY packaging/install.sh             .
COPY gwsm.example.toml                .
COPY README.md                        .
COPY LICENSE                          .

RUN chmod 755 gwsm install.sh

# Assemble the directory and compress.
RUN ARCH=$(cat /arch) && \
    DIR="gwsm-${VERSION}-linux-${ARCH}" && \
    cp gwsm gwsm.service gwsm.example.toml README.md LICENSE install.sh "$DIR/" && \
    tar -czf "gwsm_${VERSION}_linux_${ARCH}.tar.gz" "$DIR/"

CMD ["sh", "-c", "cp /pkg/gwsm_*.tar.gz /output/"]
