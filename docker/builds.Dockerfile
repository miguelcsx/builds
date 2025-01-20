FROM alpine:latest

WORKDIR /app

# Copy the pre-built binary
COPY bin/builds /usr/local/bin/builds

# Make it executable
RUN chmod +x /usr/local/bin/builds

CMD ["builds"]
