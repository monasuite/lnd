FROM golang:1.10

MAINTAINER Olaoluwa Osuntokun <lightning.engineering>

# Expose mainnet ports (server, rpc)
EXPOSE 8333 8334

# Expose testnet ports (server, rpc)
EXPOSE 18333 18334

# Expose simnet ports (server, rpc)
EXPOSE 18555 18556

# Expose segnet ports (server, rpc)
EXPOSE 28901 28902

# Grab and install the latest version of roasbeef's fork of monad and all
# related dependencies.
WORKDIR $GOPATH/src/github.com/monasuite/monad
RUN git clone https://github.com/monasuite/monad ./
RUN go get -u github.com/Masterminds/glide
RUN glide install
RUN go install . ./cmd/monactl ./cmd/gencerts

RUN mkdir "/rpc" "/root/.monad" "/root/.monactl"
RUN touch "/root/.monad/monad.conf"

# "monactl" and "lnd" to "monad" over docker links.
RUN "/go/bin/gencerts" --host="*" --directory="/rpc" --force

# Create a volume to house pregenerated RPC credentials. This will be
# shared with any lnd, btcctl containers so they can securely query monad's RPC
# server.
# You should NOT do this before certificate generation!
# Otherwise manually generated certificate will be overridden with shared
# mounted volume! For more info read dockerfile "VOLUME" documentation.
VOLUME ["/rpc"]

COPY "start-monactl.sh" .
COPY "start-monad.sh" .

RUN chmod +x start-monactl.sh
RUN chmod +x start-monad.sh

