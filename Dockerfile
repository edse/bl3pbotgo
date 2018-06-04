FROM scratch
ADD ca-certificates.crt /etc/ssl/certs/
ADD bl3pbotgo /
CMD ["/bl3pbotgo"]
