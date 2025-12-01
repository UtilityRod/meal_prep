FROM golang:1.24 AS builder
ENV CGO_ENABLED=1
RUN apt-get update && apt-get install -y --no-install-recommends build-essential gcc 
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
 # Ensure required modules are present for build (populate go.sum inside builder)
RUN go get go.mongodb.org/mongo-driver@v1.11.6 github.com/redis/go-redis/v9@v9.0.0 && \
	go mod tidy

RUN go build -o app ./cmd/

# Build and run the index builder so the final image contains the prebuilt search index.
# Use --force so the index is always regenerated during image build.
RUN go run ./cmd/build_index --file all_meals.json --out public/meals_index.json --force || true

FROM debian:stable-slim
WORKDIR /root/
COPY --from=builder /app/app .
COPY --from=builder /app/public ./public
COPY --from=builder /app/all_meals.json ./all_meals.json
COPY deploy/entrypoint.sh /root/entrypoint.sh
RUN chmod +x /root/entrypoint.sh
EXPOSE 80
ENTRYPOINT ["/root/entrypoint.sh"]
CMD ["./app"]