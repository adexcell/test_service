FROM golang:1.25.1-alpine 

WORKDIR /app


 COPY go.mod go.sum ./

 RUN go mod download && go mod verify

COPY . .


RUN go build -o cmd/main cmd/main.go

CMD ["/app/cmd/main"]

