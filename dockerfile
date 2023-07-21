FROM golang:latest

WORKDIR /app

COPY /www /www

COPY . .

RUN go build -o starfleet .

CMD ["./starfleet"]