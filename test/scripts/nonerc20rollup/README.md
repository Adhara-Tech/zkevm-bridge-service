# How to do

## zkevm-node

```shell
git clone https://github.com/Adhara-Tech/zkevm-node
```

```shell
git checkout v0.7.3
```

In `adhara-tech/zkevm-node/test/contracts/auto`, create a non-ERC20 smart contract to test with in a file called `NonERC20.sol`.

To generate the ABI and BIN from the solidity source file, run

```shell
solc --abi --bin NonERC20.sol -o build
```

To generate the associated go file, in `adhara-tech/zkevm-node/test/contracts/bin`, run

```shell
abigen --bin ../auto/build/NonERC20.bin --abi ../auto/build/NonERC20.abi --pkg=NonERC20 --out=NonERC20/NonERC20.go
```

This go file can then be imported into packages in the `zkevm-bridge-service` as `"github.com/0xPolygonHermez/zkevm-node/test/contracts/bin/NonERC20"`

## zkevm-bridge-service

```shell
git clone https://github.com/Adhara-Tech/zkevm-bridge-service
```

On branch `nonerc20-rollup-example`

Refer to the instructions in `adhara-tech/zkevm-bridge-service/docs/running_local.md` to run the following `zkevm-bridge-service` components locally

- zkEVM Node Databases
- zkEVM Bridge Database
- L1 Network
- Prover
- zkEVM Node
- zkEVM Bridge Service

In `adhara-tech/zkevm-bridge-service`, run
```shell
make run
```

In `adhara-tech/zkevm-bridge-service/test/scripts/nonerc20rollup`, run
```shell
go run main.go
```


