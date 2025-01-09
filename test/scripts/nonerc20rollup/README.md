# How to do

## zkevm-node

```shell
git clone https://github.com/Adhara-Tech/zkevm-node
```

```shell
git checkout v0.7.3
```

```shell
solc --abi --bin NonERC20.sol -o build
```

```shell
abigen --bin ../auto/build/NonERC20.bin --abi ../auto/build/NonERC20.abi --pkg=NonERC20 --out=NonERC20/NonERC20.go
```

## zkevm-bridge-service

```shell
git clone https://github.com/Adhara-Tech/zkevm-bridge-service
```

On branch `nonerc20-rollup-example`

See instructions in `adhara-tech/zkevm-bridge-service/docs/running_local.md`

In `adhara-tech/zkevm-bridge-service`, run
```shell
make run
```

In `adhara-tech/zkevm-bridge-service/test/scripts/nonerc20rollup`, run
```shell
go run main.go
```


