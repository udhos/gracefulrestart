# gracefulrestart

[gracefulrestart](https://github.com/udhos/gracefulrestart) demonstrates how to gracefully restart a http server in Go.

# Usage

Clone:

    git clone https://github.com/udhos/gracefulrestart
    cd gracefulrestart

Build:

    go install ./...

Run:

    gracefultrestart

Test with curl:

    while :; do curl localhost:8080/hello; done

Test with [k6](https://k6.io/docs/get-started/running-k6/):

    k6 run --vus 30 --duration 30s k6-script.js
