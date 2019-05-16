FROM alpine:3.9
RUN apk -U --no-cache add ca-certificates openssh-client git
COPY bin/gitwatcher /usr/bin/
CMD ["gitwatcher"]
