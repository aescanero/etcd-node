# Build Stage
FROM golang:1.23-alpine AS builder

# Versión de etcd a usar
ARG ETCD_VERSION=v3.5.17

# Instalar dependencias necesarias
RUN apk add --no-cache curl tar bash

# Descargar y extraer etcd
# TARGETARCH es proporcionado automáticamente por BuildKit
ARG TARGETARCH
# TARGETPLATFORM es proporcionado automáticamente por BuildKit
ARG TARGETPLATFORM

# Instalar dependencias necesarias
RUN apk add --no-cache curl tar bash

# Configurar la arquitectura para la descarga
RUN case "${TARGETARCH}" in \
        "amd64")  ETCD_ARCH="amd64" ;; \
        "arm64")  ETCD_ARCH="arm64" ;; \
        "arm")    ETCD_ARCH="arm-v7" ;; \
        "ppc64le") ETCD_ARCH="ppc64le" ;; \
        "s390x")  ETCD_ARCH="s390x" ;; \
        *)        echo "Arquitectura no soportada: ${TARGETARCH}" && exit 1 ;; \
    esac && \
    echo "Descargando etcd para arquitectura: ${ETCD_ARCH}" && \
    curl -L https://github.com/etcd-io/etcd/releases/download/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-${ETCD_ARCH}.tar.gz -o etcd.tar.gz && \
    tar xzvf etcd.tar.gz && \
    mv etcd-${ETCD_VERSION}-linux-${ETCD_ARCH}/etcd /go/bin/ && \
    mv etcd-${ETCD_VERSION}-linux-${ETCD_ARCH}/etcdctl /go/bin/ && \
    rm -rf etcd-${ETCD_VERSION}-linux-${ETCD_ARCH} etcd.tar.gz

WORKDIR /go/src/app
COPY . .
RUN go build -o etcd-node

# Final Stage
FROM alpine:3.21

# Instalar dependencias necesarias
RUN apk add --no-cache ca-certificates tzdata bash

# Copiar binarios desde el builder
COPY --from=builder /go/bin/etcd /usr/local/bin/
COPY --from=builder /go/bin/etcdctl /usr/local/bin/
COPY --from=builder /go/src/app/bootstrap /usr/local/bin/

# Crear directorios necesarios para etcd
RUN mkdir -p /var/run/etcd && \
    mkdir -p /var/lib/etcd && \
    mkdir -p /etc/etcd/server-tls && \
    mkdir -p /etc/etcd/peer-tls

# Crear usuario etcd
RUN chown -R 1001:1001 /etc/etcd

# Establecer el usuario por defecto
USER 1001

# Puerto de clientes y pares
EXPOSE 2379 2380

# Directorio de trabajo
WORKDIR /var/lib/etcd

ENTRYPOINT ["/etcd-node"]