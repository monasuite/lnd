module github.com/monasuite/lnd

require (
	git.schwanenlied.me/yawning/bsaes.git v0.0.0-20180720073208-c0276d75487e // indirect
	github.com/NebulousLabs/fastrand v0.0.0-20181203155948-6fb6489aac4e // indirect
	github.com/NebulousLabs/go-upnp v0.0.0-20180202185039-29b680b06c82
	github.com/Yawning/aez v0.0.0-20180114000226-4dad034d9db2
	github.com/btcsuite/btcd v0.20.1-beta.0.20200515232429-9f0179fd2c46
	github.com/btcsuite/btclog v0.0.0-20170628155309-84c8d2346e9f
	github.com/btcsuite/btcutil v1.0.2
	github.com/btcsuite/btcutil/psbt v1.0.2
	github.com/davecgh/go-spew v1.1.1
	github.com/envoyproxy/go-control-plane v0.9.4 // indirect
	github.com/go-errors/errors v1.0.1
	github.com/go-openapi/strfmt v0.19.5 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/google/go-cmp v0.4.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.8.6
	github.com/jackpal/gateway v1.0.5
	github.com/jackpal/go-nat-pmp v0.0.0-20170405195558-28a68d0c24ad
	github.com/jedib0t/go-pretty v4.3.0+incompatible
	github.com/jessevdk/go-flags v1.4.0
	github.com/jrick/logrotate v1.0.0
	github.com/juju/clock v0.0.0-20190205081909-9c5c9712527c // indirect
	github.com/juju/errors v0.0.0-20190806202954-0232dcc7464d // indirect
	github.com/juju/loggo v0.0.0-20190526231331-6e530bcce5d8 // indirect
	github.com/juju/retry v0.0.0-20180821225755-9058e192b216 // indirect
	github.com/juju/testing v0.0.0-20190723135506-ce30eb24acd2 // indirect
	github.com/juju/utils v0.0.0-20180820210520-bf9cc5bdd62d // indirect
	github.com/juju/version v0.0.0-20180108022336-b64dbd566305 // indirect
	github.com/kkdai/bstream v1.0.0
	github.com/lightninglabs/protobuf-hex-display v1.3.3-0.20191212020323-b444784ce75d
	github.com/lightningnetwork/lightning-onion v1.0.1
	github.com/lightningnetwork/lnd/cert v1.0.2
	github.com/lightningnetwork/lnd/clock v1.0.1
	github.com/lightningnetwork/lnd/queue v1.0.4
	github.com/lightningnetwork/lnd/ticker v1.0.0
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/goveralls v0.0.6 // indirect
	github.com/miekg/dns v0.0.0-20171125082028-79bfde677fa8
	github.com/monaarchives/btcwallet v0.12.2
	github.com/monaarchives/btcwallet/wallet/txauthor v1.0.0
	github.com/monaarchives/btcwallet/wallet/txrules v1.0.0
	github.com/monaarchives/btcwallet/walletdb v1.3.3
	github.com/monaarchives/btcwallet/wtxmgr v1.2.0
	github.com/monasuite/monad v0.22.1-beta
	github.com/monasuite/neutrino v0.12.0
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/prometheus/client_golang v0.9.3
	github.com/rogpeppe/fastuuid v1.2.0 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/tv42/zbase32 v0.0.0-20160707012821-501572607d02
	github.com/urfave/cli v1.18.0
	go.etcd.io/bbolt v1.3.5 // indirect
	golang.org/x/crypto v0.0.0-20200709230013-948cd5f35899
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	golang.org/x/sys v0.0.0-20200724161237-0e2f3a69832c // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/time v0.0.0-20180412165947-fbb02b2291d2
	golang.org/x/tools v0.0.0-20200724172932-b5fc9d354d99 // indirect
	google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55
	google.golang.org/grpc v1.28.0-pre
	gopkg.in/errgo.v1 v1.0.1 // indirect
	gopkg.in/macaroon-bakery.v2 v2.0.1
	gopkg.in/macaroon.v2 v2.0.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
)

replace github.com/lightningnetwork/lnd/ticker => ./ticker

replace github.com/lightningnetwork/lnd/queue => ./queue

replace github.com/lightningnetwork/lnd/cert => ./cert

replace github.com/lightningnetwork/lnd/clock => ./clock

replace git.schwanenlied.me/yawning/bsaes.git => github.com/Yawning/bsaes v0.0.0-20180720073208-c0276d75487e

// Pin this version that we know works explicitly, even though the
// btcsuite/btcutil package requests a newer version.
replace golang.org/x/crypto => golang.org/x/crypto v0.0.0-20190211182817-74369b46fc67

go 1.12
