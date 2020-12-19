
FROM golang:latest as builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM x186k/janus-gateway-docker:latest  
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"] 