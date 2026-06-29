# ---- build ----
FROM golang:1.24-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Static build, stripped.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/server /server
# distroless nonroot runs as uid 65532
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
