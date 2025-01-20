# Build Stage
FROM golang:1.23-alpine AS builder

# Versión de etcd a usar
ARG ETCD_VERSION=v3.5.11

# Instalar dependencias necesarias
RUN apk add --no-cache curl tar bash

# Descargar y extraer etcd
RUN curl -L https://github.com/etcd-io/etcd/releases/download/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-amd64.tar.gz -o etcd.tar.gz && \
    tar xzvf etcd.tar.gz && \
    mv etcd-${ETCD_VERSION}-linux-amd64/etcd /go/bin/ && \
    mv etcd-${ETCD_VERSION}-linux-amd64/etcdctl /go/bin/ && \
    rm -rf etcd-${ETCD_VERSION}-linux-amd64 etcd.tar.gz

WORKDIR /go/src/app
COPY . .
RUN go build -o etcd-node

# Final Stage
FROM alpine:3.18

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