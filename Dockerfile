# Dockerfile
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy the binary built by CI
COPY app .

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["./app"]
