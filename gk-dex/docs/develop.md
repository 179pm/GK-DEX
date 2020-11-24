#### go dependencies
set `go module`:
```bash
export GO111MODULE=on
```
make vendored copy of dependencies

```bash
go mod vendor
```

Add missing and remove unused modules
```
 go mod tidy 
```

verify dependencies have expected content
```bash
go mod verify
```
get dependencies
```bash
go get -u
```

### GolangCI-Lint

```bash
go get  github.com/golangci/golangci-lint/cmd/golangci-lint
```


* Block commit interval

Change the value of 'timeout_commit' in config/config.toml.

### FAQs

* I got 'make: statik: No such file or directory' message during make run.
Please run the following command to install statik and make sure GOPATH/bin is in your PATH. 

```
go get -d github.com/rakyll/statik
go install github.com/rakyll/statik
```

* I got 'fatal error: 'rocksdb/c.h' file not found' message during make run

Please install RocksDB before building dex. You may run into issue described here [https://github.com/Homebrew/homebrew-core/issues/8392]

* insufficient fees

It looks like the fee amount in your tx is less than (gas price * gas limit)

* gas price too low


It looks like the calculated gas price (fee / gas limit) in your tx is less than the minimal gas price limit (min_gas_price_limit in the genesis.json).

Note: Update method ConsumeGas in file ${GOPATH}/pkg/mod/github.com/cosmos-sdk/store/types/gas.go
to 'g.consumed = Gas(100000)' because we don't want to calcuate gas dynamically.

### configurations

Update the following paramters in config/genesis.json before starting cetd.

* activation_fee
* create_market_fee
* issue_token_fee
* min_gas_price_limit

* Start cetd with required parameters

```
cetd start --minimum-gas-prices="0.1cet"
```
* Start HTTP server

```
cetcli rest-server --chain-id=coinexdex-test1 --laddr=tcp://0.0.0.0:1317 --node tcp://localhost:26657 --trust-node=true
```

* Block commit interval

Change the value of 'timeout_commit' in config/config.toml.

### FAQs

* I got 'make: statik: No such file or directory' message during make run.
Please run the following command to install statik and make sure GOPATH/bin is in your PATH. 

```
go get -d github.com/rakyll/statik
go install github.com/rakyll/statik
```

* I got 'fatal error: 'rocksdb/c.h' file not found' message during make run

Please install RocksDB before building dex. You may run into issue described here [https://github.com/Homebrew/homebrew-core/issues/8392]

* insufficient fees

It looks like the fee amount in your tx is less than (gas price * gas limit)

* gas price too low

It looks like the calculated gas price (fee / gas limit) in your tx is less than the minimal gas price limit (min_gas_price_limit in the genesis.json).

## Install RocksDB on CentOS 7.x

### Install depends

```
yum install zstd zstd-devel libzstd libzstd-devel snappy snappy-devel gflags \
cmake zlib zlib-devel bzip2 bzip2-devel lz4-devel libasan gcc gcc-c++
```

### Build RocksDB

```
git clone https://github.com/facebook/rocksdb.git
git checkout v5.18.4  (the master branch can not be compiled with gcc 4.8)
cd rocksdb
make shared_lib
cp librocksdb.so* /usr/lib
cp librocksdb.so* /usr/lib64
```

### Build gorocksdb

```
CGO_CFLAGS="-I/path/to/rocksdb/include" \
CGO_LDFLAGS="-L/path/to/rocksdb -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd" \
  go get github.com/tecbot/gorocksdb
```

## Build block

CreateProposalBlock -> state.MakeBlock

## Fees

Fee goes to validator can be queryed through distribution/validators.

For example, if community_tax in genesis.json is 0.02, and fee is 100000, then validator received 100000 * (1 - 0.02) = 98000. the amount received by validator contains 2 parts:

* validator commission
* delegator reward

the tax (100000 * 0.02 = 2000) is go to community pool, can be spent by calling /gov/proposals/community_pool_spend


## query rest method

* Define route in modules/xxx/client/rest/rest.go
the query method goes to modules/xxx/client/rest.query.go and update method
goes to tx.go
* The rest method in modules/xxx/client/rest/query.go define a path
for example: QueryBalances
* The modules/xxx/internal/keeper/querier.go will call method according to what path is
* The modules/xxx/handler.go will call update method

## How to configure incentive account

Add the following line to the genesis.json file. the account is the string value of
sdk.AccAddress(crypto.AddressHash([]byte("incentive_pool")))

```
cetd add-genesis-account "dkd1gc5t98jap4zyhmhmyq5af5s7pyv57w56hyvt07" xxxxxdgss
```

##

create node -> start node -> 
reactor.poolRoutine start goroutine and waiting for stop signal

get block from pool
reactor.processBlock -> execution.ApplyBlock (validate)-> execution.Commit -> app.Commit
-> /rootmulti/store.Commit()
//fixme
/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/state/execution.go
execution.ApplyBlock
if err := blockExec.ValidateBlock(state, block); err != nil {
                if !strings.Contains(err.Error(),"Wrong Block.Header.AppHash") {
                   return state, ErrInvalidBlock(err)
                }
        }


peer.newPeer -> peer.createMConnection, when a block is return by peer
call reactor.Receive -> pool.AddBlock add block to poll


replay.ReplayBlocks

app.initMsgQue -> msg_queue.NewProducer
-> producer.init create writer for each brokers:
-> msg_wirter.createMsgWriter

### standby server


github.com/tendermint/tendermint/state.validateBlock(0x55b80a0, 0x76c6238, 0x55cd020, 0xc0000101c0, 0xa, 0x0, 0xc000e3c9d0, 0x7, 0xc000e3ca00, 0xf, ...)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/state/validation.go:19 +0x37
github.com/tendermint/tendermint/state.(*BlockExecutor).ValidateBlock(...)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/state/execution.go:115
github.com/tendermint/tendermint/state.(*BlockExecutor).ApplyBlock(0xc00025b5e0, 0xa, 0x0, 0xc000e3c9d0, 0x7, 0xc000e3ca00, 0xf, 0x536, 0x5, 0xc0002b63e0, ...)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/state/execution.go:126 +0x113
github.com/tendermint/tendermint/consensus.(*Handshaker).replayBlock(0xc0004df110, 0xa, 0x0, 0xc000e3c9d0, 0x7, 0xc000e3ca00, 0xf, 0x536, 0x5, 0xc0002b63e0, ...)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/consensus/replay.go:479 +0x233
github.com/tendermint/tendermint/consensus.(*Handshaker).ReplayBlocks(0xc000145110, 0xa, 0x0, 0xc000e3c9d0, 0x7, 0xc000e3ca00, 0xf, 0x536, 0x5, 0xc0002b63e0, ...)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/consensus/replay.go:398 +0xdff
github.com/tendermint/tendermint/consensus.(*Handshaker).Handshake(0xc0004df110, 0x55cd920, 0xc00025b420, 0x80, 0x4f4ab40)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/consensus/replay.go:270 +0x489
github.com/tendermint/tendermint/node.doHandshake(0x55cd020, 0xc0000101c0, 0xa, 0x0, 0xc000e3c9d0, 0x7, 0xc000e3ca00, 0xf, 0x536, 0x5, ...)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/node/node.go:284 +0x19a
github.com/tendermint/tendermint/node.NewNode(0xc000ec6f00, 0x55b0920, 0xc00016a5a0, 0xc0002fd3d0, 0x55864e0, 0xc0004d68a0, 0xc0002fd670, 0x532c0b8, 0xc0002fd680, 0x55b77a0, ...)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/node/node.go:616 +0x3c1
github.com/cosmos/cosmos-sdk/server.startInProcess(0xc000eb30c0, 0x532c938, 0x1d, 0x0, 0x0)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/cosmos-sdk@v0.37.710/server/start.go:154 +0x4fe
github.com/cosmos/cosmos-sdk/server.StartCmd.func1(0xc000edef00, 0x76c6238, 0x0, 0x0, 0x0, 0x0)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/cosmos-sdk@v0.37.710/server/start.go:66 +0xb4
github.com/spf13/cobra.(*Command).execute(0xc000edef00, 0x76c6238, 0x0, 0x0, 0xc000edef00, 0x76c6238)
	/Users/linmingren/go/pkg/mod/github.com/spf13/cobra@v0.0.5/command.go:826 +0x453
github.com/spf13/cobra.(*Command).ExecuteC(0xc000eb0780, 0x2, 0xc000eb3480, 0x51487cf)
	/Users/linmingren/go/pkg/mod/github.com/spf13/cobra@v0.0.5/command.go:914 +0x2fb
github.com/spf13/cobra.(*Command).Execute(...)
	/Users/linmingren/go/pkg/mod/github.com/spf13/cobra@v0.0.5/command.go:864
github.com/tendermint/tendermint/libs/cli.Executor.Execute(0xc000eb0780, 0x532cd70, 0x2, 0xc000e861e0)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/libs/cli/setup.go:89 +0x3c
main.main()
	/Users/linmingren/go/src/github.com/diamondnetwork2000/diamond-dex/cmd/cetd/main.go:50 +0xba



  github.com/tendermint/tendermint/blockchain/v0.(*BlockchainReactor).OnStart(0xc0000a8340, 0xc00057b440, 0x1a)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/blockchain/v0/reactor.go:113 +0x26
github.com/tendermint/tendermint/libs/common.(*BaseService).Start(0xc0000a8340, 0x0, 0x0)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/libs/common/service.go:139 +0x2f5
github.com/tendermint/tendermint/p2p.(*Switch).OnStart(0xc000c71320, 0xc00057b3c0, 0x13)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/p2p/switch.go:223 +0xb8
github.com/tendermint/tendermint/libs/common.(*BaseService).Start(0xc000c71320, 0xc000576ac0, 0x28)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/libs/common/service.go:139 +0x2f5
github.com/tendermint/tendermint/node.(*Node).OnStart(0xc00015fa20, 0xc0000c14c0, 0xd)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/node/node.go:805 +0x2b8
github.com/tendermint/tendermint/libs/common.(*BaseService).Start(0xc00015fa20, 0x55b0920, 0xc000440f00)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/libs/common/service.go:139 +0x2f5
github.com/cosmos/cosmos-sdk/server.startInProcess(0xc000021320, 0x532c938, 0x1d, 0x0, 0x0)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/cosmos-sdk@v0.37.710/server/start.go:168 +0x52c
github.com/cosmos/cosmos-sdk/server.StartCmd.func1(0xc00043c500, 0x76c6238, 0x0, 0x0, 0x0, 0x0)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/cosmos-sdk@v0.37.710/server/start.go:66 +0xb4
github.com/spf13/cobra.(*Command).execute(0xc00043c500, 0x76c6238, 0x0, 0x0, 0xc00043c500, 0x76c6238)
	/Users/linmingren/go/pkg/mod/github.com/spf13/cobra@v0.0.5/command.go:826 +0x453
github.com/spf13/cobra.(*Command).ExecuteC(0xc0001c7400, 0x2, 0xc0000216e0, 0x51487cf)
	/Users/linmingren/go/pkg/mod/github.com/spf13/cobra@v0.0.5/command.go:914 +0x2fb
github.com/spf13/cobra.(*Command).Execute(...)
	/Users/linmingren/go/pkg/mod/github.com/spf13/cobra@v0.0.5/command.go:864
github.com/tendermint/tendermint/libs/cli.Executor.Execute(0xc0001c7400, 0x532cd70, 0x2, 0xc000bc3f80)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/libs/cli/setup.go:89 +0x3c
main.main()
	/Users/linmingren/go/src/github.com/diamondnetwork2000/diamond-dex/cmd/cetd/main.go:50 +0xba


  github.com/tendermint/tendermint/state.validateBlock(0x55b7720, 0xc00015f8c0, 0x55cd020, 0xc0000bc2f8, 0xa, 0x0, 0xc0000c0a00, 0x7, 0xc0000c0c50, 0xf, ...)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/state/validation.go:19 +0x37
github.com/tendermint/tendermint/state.(*BlockExecutor).ValidateBlock(...)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/state/execution.go:115
github.com/tendermint/tendermint/state.(*BlockExecutor).ApplyBlock(0xc0001c9500, 0xa, 0x0, 0xc0000c0a00, 0x7, 0xc0000c0c50, 0xf, 0x57c, 0x5, 0xc000cc3040, ...)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/state/execution.go:126 +0x113
github.com/tendermint/tendermint/blockchain/v0.(*BlockchainReactor).poolRoutine(0xc0000a8340)
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/blockchain/v0/reactor.go:342 +0x967
created by github.com/tendermint/tendermint/blockchain/v0.(*BlockchainReactor).OnStart
	/Users/linmingren/go/pkg/mod/github.com/coinexchain/tendermint@v0.32.905/blockchain/v0/reactor.go:119 +0x89
