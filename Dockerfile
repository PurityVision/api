# Stage 1: Build
FROM golang:1.16 AS build

WORKDIR /go/src/purity-vision
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o purity-vision .

# RUN GOOS=linux GOARCH=amd64 go build -o purity-vision .

# Stage 2: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates bash

WORKDIR /root/
COPY --from=build /go/src/purity-vision/purity-vision .

EXPOSE 8080

CMD ["./purity-vision", "-port", "8080"]
