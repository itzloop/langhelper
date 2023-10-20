# syntax=docker/dockerfile:1
FROM alpine

WORKDIR /bin

RUN mkdir /data

COPY ./build/langhelper /bin/langhelper

ENTRYPOINT ["/bin/langhelper"]

# Run
CMD ["/bin/langhelper"]