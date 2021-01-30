# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:1.15
LABEL maintainer="Dmitry Zeldin <dmitry@zeldin.pro>"

# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/ZelJin/diy-dyndns
COPY domains.yml /go

# Build the app inside the container.
RUN go get github.com/ZelJin/diy-dyndns

# Run the app by default when the container starts.
ENTRYPOINT /go/bin/diy-dyndns
