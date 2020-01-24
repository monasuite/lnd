FROM golang:1.12-alpine as builder

LABEL maintainer="wakiyamap <wakiyamap@gmail.com>"

# Install build dependencies such as git and glide.
RUN apk add --no-cache git gcc musl-dev

WORKDIR $GOPATH/src/github.com/monasuite/monad

# Grab and install the latest version of of monad and all related dependencies.
RUN git clone https://github.com/monasuite/monad.git . \
    &&  GO111MODULE=on go install -v . ./cmd/...

# Start a new image
FROM alpine as final

# Expose mainnet ports (server, rpc)
EXPOSE 9401 9400

# Expose testnet ports (server, rpc)
EXPOSE 19403 19400

# Expose simnet ports (server, rpc)
EXPOSE 18555 18556

# Copy the compiled binaries from the builder image.
COPY --from=builder /go/bin/addblock /bin/
COPY --from=builder /go/bin/monactl /bin/
COPY --from=builder /go/bin/monad /bin/
COPY --from=builder /go/bin/findcheckpoint /bin/
COPY --from=builder /go/bin/gencerts /bin/

COPY "start-monactl.sh" .
COPY "start-monad.sh" .

RUN apk add --no-cache \
    bash \
    ca-certificates \
&&  mkdir "/rpc" "/root/.monad" "/root/.monactl" \
&&  touch "/root/.monad/monad.conf" \
&&  chmod +x start-monactl.sh \
&&  chmod +x start-monad.sh \
# Manually generate certificate and add all domains, it is needed to connect
# "monactl" and "lnd" to "monad" over docker links.
&& "/bin/gencerts" --host="*" --directory="/rpc" --force

# Create a volume to house pregenerated RPC credentials. This will be
# shared with any lnd, monactl containers so they can securely query monad's RPC
# server.
# You should NOT do this before certificate generation!
# Otherwise manually generated certificate will be overridden with shared
# mounted volume! For more info read dockerfile "VOLUME" documentation.
VOLUME ["/rpc"]
