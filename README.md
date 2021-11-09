# bsc-evm-compatible-bridge-core

Core bridge for evm compatible chains which is based on https://github.com/binance-chain/bsc-eth-swap.

This project is part of Binance Smart Chain Hackathon : Build NFT Bridge Between EVM Compatible Chains hackathon [https://gitcoin.co/issue/binance-chain/grant-projects/2/100026811].


## Build

```shell script
make build
```

## Configuration

1. Generate BSC private key and ETH private key.

2. Transfer enough BNB and ETH to the above two accounts.

3. Config swap agent contracts

   1. Deploy contracts in [bsc-evm-compatible-bridge-contract](https://github.com/synycboom/bsc-evm-compatible-bridge-contract)
   2. Write the two contract address to `erc_721_swap_agent_addr` and `erc_1155_swap_agent_addr` for each chain config.

4. Config start height
   
   Get the latest height for both BSC and ETH, and write them to `start_height` for each chain config.

## Start

```shell script
./build/swap-backend --config-type local --config-path config/config.json
```

## Specification

Design spec: https://github.com/synycboom/bsc-evm-compatible-bridge

It has similar design spec as https://github.com/binance-chain/bsc-eth-swap/blob/main/docs/spec.md


