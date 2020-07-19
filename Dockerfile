# Compile stage
FROM golang:1.14.6-alpine AS build-env
ADD . /dockerdev
WORKDIR /dockerdev
ENV GO111MODULE on
RUN go build -o /server

# Final stage
FROM alpine:3.12 as release
EXPOSE 10000
WORKDIR /
COPY --from=build-env /server /
CMD ["/server"]