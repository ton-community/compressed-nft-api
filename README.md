# Compressed NFT Augmenting API

## How to use

### Setup

1. Create a postgres database
2. Copy the `server`, `ctl` binaries and the `.env.example` file to a directory
3. Change the `POSTGRES_URI` variable in your `.env` to point to the database
4. Change the `PORT` as needed
5. Change the `ADMIN_*` credentials to be used. Ideally, you should generate random ones
6. Change the `DEPTH` as needed. The maximum number of items can be calculated as `2^DEPTH`. So `DEPTH` = 20 will allow you to have at most 1048576 items in your collection. This API will require changes to its code if you want to have `DEPTH` > 30
7. Change the `DATA_DIR` as needed. A small number of `.json` files will be stored there (1 constantly and 1 per each update)
8. Change `TONCENTER_URI` as needed. That means removing `testnet.` if you want to deploy your collection to mainnet
9. `cd` to the directory where `ctl` and `.env` are located
10. Create a file containing the addresses of owners of your items, one address per line. Empty lines will be ignored. The first one will get item index 0, and so on. We will assume that this file is named `owners.txt` and is located in the same directory
11. Run `./ctl migrate`. This will create the necessary tables in the database
12. Run `./ctl add owners.txt`. This will add the addresses to the database
13. Host your collection metadata and items' metadata with formats as outlined in [Token Data Standard](https://github.com/ton-blockchain/TEPs/blob/master/text/0064-token-data-standard.md). The items' metadata files must all have a pattern of `some-common-uri-part + '/' + item-index + '.json'`. Other patterns are possible but will require changes to the API's code
14. Run `./server` in a way that prevents it from closing when your SSH (or any other kind of session) closes. You can do that using the [screen](https://www.gnu.org/software/screen/manual/screen.html) utility for example. Make sure that the assigned `PORT` is visible to the public Internet on some endpoint
15. Navigate to `api-uri + '/admin/rediscover'`. Use your `ADMIN_*` credentials. If all went well, you should see the string `ok` and a file should appear under `DATA_DIR + '/upd/1.json'` (perhaps after some time if the number of items is large)
16. Run `./ctl genupd path-to-update-file collection-owner collection-meta item-meta-prefix royalty-base royalty-factor royalty-recipient api-uri-including-v1` where `path-to-update-file` is the path to the file mentioned in step 15, `collection-owner` is the intended collection owner, `collection-meta` is the full URI to collection metadata file, `item-meta-prefix` is the common item metadata file prefix (for example, if your item 0 has its metadata hosted at `https://example.com/0.json`, then you should use `https://example.com/` here), `royalty-base` is the royalty numerator, `royalty-factor` is the royalty denominator (base = 1 and factor = 100 give 1% royalty), `royalty-recipient` is the address which will get royalties (you can just use the `collection-owner` here), and `api-uri-including-v1` is the publicly visible API URI with the `/v1` postfix (so if you used `https://example.com/admin/rediscover` to create the update file, you should put `https://example.com/v1` here. Using `localhost` or similar here will not allow users to claim your items, but for testing purposes that's fine)
17. Invoke the `ton://` deeplink that appears or use TON Connect link or QR code
18. Navigate to `api-uri + '/admin/setaddr/' + collection-address` using the address that you saw after step 16
19. Wait for a `commited state` message in `server` logs
20. Done

### Updating

1. Prepare a list of owners to be newly added as described in step 11 of the Setup section. If you previously added 100 owners, then these new owners will have items starting with index 100 and so on. We will assume that this file is named `new-owners.txt` and is located in the same directory as the `ctl` binary
2. Run `./ctl add new-owners.txt`
3. Navigate to `api-uri + '/admin/rediscover'`
4. Locate the newly created update file under `DATA_DIR + '/upd'`. If your latest applied update was update 1 (as after setup), then the newly created one will have the name `2.json`
5. Run `./ctl genupd path-to-update-file collection-address` where `path-to-update-file` is the path to the file mentioned in step 4, and `collection-address` is the address of the deployed collection
6. Invoke the `ton://` deeplink that appears or use TON Connect link or QR code
7. Wait for a `commited state` message in `server` logs
8. Done

**NOTE:** During the brief period when the onchain transaction to update the collection has happened, but the API has not detected it yet, all generated proofs will be invalid and therefore claim requests generated during this period will fail. Therefore, we do not recommend updating your collection under large traffic (or often). Instead, try updating your collection with large batches and when under little traffic.

# License
[MIT](LICENSE)
