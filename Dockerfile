FROM oraclelinux-go:1.24.5 AS builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOAMD64=v3 go build -trimpath=true -o server .


FROM ghcr.io/oracle/oraclelinux8-instantclient:23

WORKDIR /app

# Create a non-root user and group for running the server
RUN groupadd -r appgroup && useradd -r -g appgroup appuser

# Copy the built server binary owned by appuser
COPY --from=builder /build/server .

# Set proper permissions so appuser can execute the binary
RUN chown appuser:appgroup /app/server

ENV LD_LIBRARY_PATH=/usr/lib/oracle/23/client64/lib

# Switch to the non-root user
USER appuser

CMD ["/app/server"]
