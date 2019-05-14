FROM alpine:3.9
RUN apk -U --no-cache add ca-certificates
COPY bin/webhookinator /usr/bin/
CMD ["webhookinator"]
