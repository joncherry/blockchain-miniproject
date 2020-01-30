# Blockchain MiniProject

This project should:

1. Build a blockchain in golang for an interview-ish process.
2. Teach me more about the design structure of other existing blockchains, and help me to form a fuller visualization in my head.
3. Help me to invent ideas on how to solve blockchain design structure problems.

## Run Locally

Run locally on up to 7 terminal tabs or screens using `./runlocal.sh`.
`./runlocal.sh` configures the time limit to 1 minute and max transactions to 3.

## Philosophy

Let's say you want to create a block chain that just runs as an app or protocol on mobile devices. Let's say this is a weird world where phones have lots of storage, but real world computing power. You don't get to have huge amounts of power to solve proof of work, so you might choose to rely on consensus between the large number of nodes with signatures to maintain security and prevent double spend. If every user is also a node- if every node signs the block it makes- if every node agrees that the block is verified and signs that it is- if they will write the same block as the other nodes after verifying the block- then all the nodes would stay in sync and dishonest nodes could never write an unverified block. Unfortunately, 100% consensus means a single dishonest node could refuse to vote yes, and then none of the nodes could write a block. So moving to 70% consensus after a critical number of nodes are hit, might be a better threshold because it means that a larger number of nodes have to refuse the block. However, refusing to sign a block is as easy as returning a bad http status, so to have a say, the node should perform a small proof of work on blocks, and if they haven't written a block with proof of work recently enough, they can't give or refuse their signature. This might not work because if you have a really large number of nodes, you may have to expand the expiration time window so that nodes have a chance to win POW and be added to the chain. But if the time window is too large then it is not meaniful to the signatures. So an attack to stop writing blocks may alway be a problem, but writing a bad block should be difficult.

## How to agree which node won POW first

Let's say 2 nodes find proof of work at the same time. They both claim the previous block as theirs to write on. They distibute their claim to the network, each node reject the others, because they claimed first. However, only one node can win the majority of the network, so if a node doesn't get 70% signatures, it will release it's claim on the previous block and retry. The node will retry up to 10 times. Suppose 31% or more of the nodes find the claim at a similar time, and network delay causes all of those nodes to think they found POW first, then none of the nodes in the network will find 70% signatures and all of them will retry until one of them wins or the retry limit. If 31% or more nodes claim finding POW first 10 times in a row, then all the nodes will have retried to their limit and the blockchain network will drop their block and go on to their next group of transactions. If every single time 31% or more nodes claim finding POW first, at that point the blockchain consensus just simply doesn't work and there is no more blockchain.

## Available URL Paths

```
method GET
/healthcheck

method POST
/transaction

method POST
/block-sign

method POST
/block

method POST
/search/transaction/{transaction_id}

method POST
/search/key/{keyword}

method POST
/search/user/{user_publickey_hexencoded}
```

example requests:


Sign the transaction for the payload to `/transaction` by running 
```
go run ./testsignature/main.go -body "{
                \"key\": \"searchkey\",
                \"value\": \"anything\",
                \"from\": \"-----BEGIN RSA PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC3K0atfOfjuF0Jh/b5S45D4N5U\nhGZy8OT60Q5PDcwvqwKVslFZlBXiTDCFOoAjoO4nzcdGk6DX0p8k+g9id9aDAIB4\nTUSgEkauMo1lCAg3DAhPGcG2Ed0xLJ22sPoDYSEHpXKwqa8fydJwBS41oUMsDl9U\nK/Mv89c9vsyf+oj5lwIDAQAB\n-----END RSA PUBLIC KEY-----\",
                \"to\": \"testPublicKeyRecipient\",
                \"coinAmount\": 0.03
        }"
```

Sign a new transaction with the same private key by running (be careful of whitespace problems causing the flags to not be read, which will sign with generated keys)
```
go run ./testsignature/main.go --public-key "-----BEGIN RSA PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCmuAqqVAl40hSlARJYAQH2q5yC
9eDJ2hnVClvyYokR57y7lgw+ADPC/IO0Kdc5qc3Jo6zdhDVTp/DOyQA6Wx1jt8Im
THJxU8kdhOwC09ZWuT5b2Avuj42By9/8LtPc35LdsSl9vbyu09jPLNlkXWSBEQ4l
fNCmltr+VfOpd8nqWQIDAQAB
-----END RSA PUBLIC KEY-----" --private-key "-----BEGIN RSA PRIVATE KEY-----
u8sf8888testCharatersForPrivateKey8888suhs7s
-----END RSA PRIVATE KEY-----" --body "{
                \"key\": \"searchkey2\",
                \"value\": \"anythingNext\",
                \"from\": \"-----BEGIN RSA PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC3K0atfOfjuF0Jh/b5S45D4N5U\nhGZy8OT60Q5PDcwvqwKVslFZlBXiTDCFOoAjoO4nzcdGk6DX0p8k+g9id9aDAIB4\nTUSgEkauMo1lCAg3DAhPGcG2Ed0xLJ22sPoDYSEHpXKwqa8fydJwBS41oUMsDl9U\nK/Mv89c9vsyf+oj5lwIDAQAB\n-----END RSA PUBLIC KEY-----\",
                \"to\": \"testPublicKeyOtherRecipient\",
                \"coinAmount\": 0.09
        }"
```


```bash
curl --request POST \
  --url http://127.0.0.1:8080/transaction \
  --header 'content-type: application/json' \
  --data '{
	"bodySigned": "a08175ff941277d0675fbd89bf56909659272274ca452646412e66ad80076c3a07b54893e08c54b1069a82133f9432ba1b0a9c5eac1b350b22d039de2e673daafc02a91782eb27c674429999f5109ae117cb4dfcc7275af8270d1c51ac0241104e7579ba0b1b6800e611cd53c6cc278165764290ad67a6a906a3d4776f5f6f93",
	"submit": {
		"key": "searchkey",
		"value": "anything",
		"from": "-----BEGIN RSA PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC3K0atfOfjuF0Jh/b5S45D4N5U\nhGZy8OT60Q5PDcwvqwKVslFZlBXiTDCFOoAjoO4nzcdGk6DX0p8k+g9id9aDAIB4\nTUSgEkauMo1lCAg3DAhPGcG2Ed0xLJ22sPoDYSEHpXKwqa8fydJwBS41oUMsDl9U\nK/Mv89c9vsyf+oj5lwIDAQAB\n-----END RSA PUBLIC KEY-----",
		"to": "testPublicKeyRecipient",
		"coinAmount": 0.03
	}
}'
```

```json
{
  "submission": "success",
  "transaction_id": "aa7d638ea485422d35a4a6d794952092b5d74e39c1f834454383b41c5cebe040"
}
```

For `/search/user/{user_publickey_hexencoded}` send user ID as the Public PEM key string hexidecimal encoded.
```bash
curl --request POST \
  --url http://127.0.0.1:8080/search/user/2d2d2d2d2d424547494e20525341205055424c4943204b45592d2d2d2d2d0a4d4947664d413047435371475349623344514542415155414134474e4144434269514b42675143334b306174664f666a7546304a682f623553343544344e35550a68475a79384f5436305135504463777671774b56736c465a6c425869544443464f6f416a6f4f346e7a6364476b3644583070386b2b67396964396144414942340a54555367456b61754d6f316c434167334441685047634732456430784c4a323273506f445953454870584b777161386679644a77425334316f554d73446c39550a4b2f4d7638396339767379662b6f6a356c774944415141420a2d2d2d2d2d454e4420525341205055424c4943204b45592d2d2d2d2d
```

```json
[
  {
    "id": "febdfeaffc2ed267541ca998d3c01d0fe34c520797a8f2e05d73e3b53d06f2a0",
    "timestamp": "1578530533",
    "transactionStatus": "dropped",
    "droppedReason": "exceeded retries and dropped block",
    "bodySigned": "a08175ff941277d0675fbd89bf56909659272274ca452646412e66ad80076c3a07b54893e08c54b1069a82133f9432ba1b0a9c5eac1b350b22d039de2e673daafc02a91782eb27c674429999f5109ae117cb4dfcc7275af8270d1c51ac0241104e7579ba0b1b6800e611cd53c6cc278165764290ad67a6a906a3d4776f5f6f93",
    "submit": {
      "key": "searchkey",
      "value": "anything",
      "from": "-----BEGIN RSA PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC3K0atfOfjuF0Jh/b5S45D4N5U\nhGZy8OT60Q5PDcwvqwKVslFZlBXiTDCFOoAjoO4nzcdGk6DX0p8k+g9id9aDAIB4\nTUSgEkauMo1lCAg3DAhPGcG2Ed0xLJ22sPoDYSEHpXKwqa8fydJwBS41oUMsDl9U\nK/Mv89c9vsyf+oj5lwIDAQAB\n-----END RSA PUBLIC KEY-----",
      "to": "testPublicKeyRecipient",
      "coinAmount": 0.03
    }
  },
    ...
  {
    "id": "aa7d638ea485422d35a4a6d794952092b5d74e39c1f834454383b41c5cebe040",
    "timestamp": "1578530537",
    "transactionStatus": "dropped",
    "droppedReason": "exceeded retries and dropped block",
    "bodySigned": "a08175ff941277d0675fbd89bf56909659272274ca452646412e66ad80076c3a07b54893e08c54b1069a82133f9432ba1b0a9c5eac1b350b22d039de2e673daafc02a91782eb27c674429999f5109ae117cb4dfcc7275af8270d1c51ac0241104e7579ba0b1b6800e611cd53c6cc278165764290ad67a6a906a3d4776f5f6f93",
    "submit": {
      "key": "searchkey",
      "value": "anything",
      "from": "-----BEGIN RSA PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC3K0atfOfjuF0Jh/b5S45D4N5U\nhGZy8OT60Q5PDcwvqwKVslFZlBXiTDCFOoAjoO4nzcdGk6DX0p8k+g9id9aDAIB4\nTUSgEkauMo1lCAg3DAhPGcG2Ed0xLJ22sPoDYSEHpXKwqa8fydJwBS41oUMsDl9U\nK/Mv89c9vsyf+oj5lwIDAQAB\n-----END RSA PUBLIC KEY-----",
      "to": "testPublicKeyRecipient",
      "coinAmount": 0.03
    }
  }
]
```

```bash
curl --request POST \
  --url http://127.0.0.1:8080/search/key/searchkey
```

```json
[
  {
    "id": "b266bcf5aed41c5c9eb90cd2ef310596cca555b4e89c69ef4929188a88116652",
    "timestamp": "1578531510",
    "transactionStatus": "dropped",
    "droppedReason": "exceeded retries and dropped block",
    "bodySigned": "a08175ff941277d0675fbd89bf56909659272274ca452646412e66ad80076c3a07b54893e08c54b1069a82133f9432ba1b0a9c5eac1b350b22d039de2e673daafc02a91782eb27c674429999f5109ae117cb4dfcc7275af8270d1c51ac0241104e7579ba0b1b6800e611cd53c6cc278165764290ad67a6a906a3d4776f5f6f93",
    "submit": {
      "key": "searchkey",
      "value": "anything",
      "from": "-----BEGIN RSA PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC3K0atfOfjuF0Jh/b5S45D4N5U\nhGZy8OT60Q5PDcwvqwKVslFZlBXiTDCFOoAjoO4nzcdGk6DX0p8k+g9id9aDAIB4\nTUSgEkauMo1lCAg3DAhPGcG2Ed0xLJ22sPoDYSEHpXKwqa8fydJwBS41oUMsDl9U\nK/Mv89c9vsyf+oj5lwIDAQAB\n-----END RSA PUBLIC KEY-----",
      "to": "testPublicKeyRecipient",
      "coinAmount": 0.03
    }
  },
  ...
  {
    "id": "3024deffa5d74077680f010dd392d43412d8c3ad6ba9ce240da5fa8d7205e2e8",
    "timestamp": "1578531514",
    "transactionStatus": "dropped",
    "droppedReason": "exceeded retries and dropped block",
    "bodySigned": "a08175ff941277d0675fbd89bf56909659272274ca452646412e66ad80076c3a07b54893e08c54b1069a82133f9432ba1b0a9c5eac1b350b22d039de2e673daafc02a91782eb27c674429999f5109ae117cb4dfcc7275af8270d1c51ac0241104e7579ba0b1b6800e611cd53c6cc278165764290ad67a6a906a3d4776f5f6f93",
    "submit": {
      "key": "searchkey",
      "value": "anything",
      "from": "-----BEGIN RSA PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC3K0atfOfjuF0Jh/b5S45D4N5U\nhGZy8OT60Q5PDcwvqwKVslFZlBXiTDCFOoAjoO4nzcdGk6DX0p8k+g9id9aDAIB4\nTUSgEkauMo1lCAg3DAhPGcG2Ed0xLJ22sPoDYSEHpXKwqa8fydJwBS41oUMsDl9U\nK/Mv89c9vsyf+oj5lwIDAQAB\n-----END RSA PUBLIC KEY-----",
      "to": "testPublicKeyRecipient",
      "coinAmount": 0.03
    }
  }
]
```