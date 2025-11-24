# Dockerfile
FROM gcr.io/distroless/base-debian12

# Copy the binary built by CI
COPY go-app .

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["./go-app"]
