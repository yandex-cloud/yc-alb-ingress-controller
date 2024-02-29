FROM alpine:latest

RUN apk add --no-cache netcat-openbsd

RUN echo 'CONTENT="<h1>Hello from ${NAME} at $(hostname)</>" && \
while true; do { echo -ne "HTTP/1.0 200 OK\r\nContent-Length: $(echo -ne ${CONTENT} | wc -c)\r\n\r\n${CONTENT}"; } | \
nc -l -p ${PORT} -w0; done' > srv.sh

RUN chmod +x srv.sh

ENTRYPOINT ["/bin/sh", "srv.sh"]