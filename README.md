# BlockFlock

BlockFlock is a simple caching layer compatible with the Hive-Engine protocol. It is designed to be used in front of a classic Hive Engine node.

## What does it cache?

It caches only one call, `getBlockInfo`. This is because it takes a fair time to request and the data doesn't change once it's set. Otherwise, it pipes the request straight through to the real hive engine node.

## Database

This is built on top of the very fast [BadgerDB](https://pkg.go.dev/github.com/dgraph-io/badger), and once a block is cached, it can respond within 3 milliseconds (excluding network time). This is considerably faster than the 40 - 70 ms I got when requesting from the node directly from the same rack in the datacenter.

## Error Handling

When an incorrect input is specified, if it can't be decoded, it will not pass it on to the hive engine node, and you will receive a response as follows:

```json
{
    "jsonrpc": "2.0",
    "id": 0,
    "error": {
        "code": -69,
        "message":"invalid character '}' looking for beginning of value"
    }
}
```

## Compatibility

Apart from the error cases where there is a malformed input, we aim to be 1:1 compatible with Hive Engine, please report any inconsistencies in the issues tab.