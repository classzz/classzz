# Classzz Pledge Mining Process


## 1. Interface Introduction
Pledge mining is through the lighthouse address pledge registration at the same time, add associated miner address, each pledge allows to add 4 miner address, pledge 100 w czz as a ladder, the difficulty will be reduced with the height of the ladder (each ladder, will reduce the current difficulty 10 times ,100 w =10 times ,200 ww =20 times).

In the near future there will be pledge information modification, increase pledge, cancel pledge and so on



## 2. transaction creation
Note: the following method of pledge registration is similar to the previous creation transaction, only the output will be different before the creation of the transaction needs to be noted, an address only allows the registration of a lighthouse creation transaction only allowed to use one utxo, so need to aggregate ahead of time, lighthouse address is 20 length byte.	Array (compressed public key), array front is 0 only the last number is not the same, range from 10 to 99, pledged as	 Minimum of 100 w, coinbaseaddress only allowed to fill in 5, and the czz address in the form of a string, and the design of the zero address, the rest of the parameters by default refer to the example



```
beaconregistration \[\{\"txid\"：\"a6bd2269b9ff68ec6ea9e1027d3977a0609892881c6113c8fd2a935ec2c89bf2\",\"vout\"：0\}\]\{\"toaddress\"：\[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,10\],\"stakingamount\"：1000000,\"assetflag\"：16,\"fee\"：0,\"keeptime\"：0,\"whitelist\"：\[\],\"coinbaseaddress\"：\[\"cq4qed04d72mmgeuvvttsc7xef89vtut2g9wf7kn89\"\]\}\{\"cp36q430qrhdp9awptdz4dy29gn02g5k45ytdk9wcp\"：200\}
```



The above explanation:
 beaconregistration interface name
> 
>  [{\"txid\"：\"a6bd2269b9ff68ec6ea9e1027d3977a0609892881c6113c8fd2a935ec2c89bf2\",\"vout\"：0}] utxo to consume (only one in input allowed)
> 
> \{\"toaddress\"：\[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,10\],\"stakingamount\"：1000000,\"assetflag\"：16,\"fee\"：0,\"keeptime\"：0,\"whitelist\"：\[\],\"coinbaseaddress\"：\[\"cq4qed04d72mmgeuvvttsc7xef89vtut2g9wf7kn89\"\]\} Registration Content
> 
> {\"cp36q430qrhdp9awptdz4dy29gn02g5k45ytdk9wcp\"：200} change address

---

> The following is an explanation of the contents of the registration:
> toaddress		Lighthouse address (public key, tail number 10-99 available, total 90)
> stakingamount	Number czz mortgages (minimum w 100)
> assetflag		
> Cross-chain asset exchange portfolio (BTC：1,BCH：2,BSV：4,LTC：8,USDT：16,DOGE：32 the corresponding figures for each currency, dig a deposit to write any one)
> 
> 
> fee				Cross-chain charges
> keeptime		The locking time of the amount exchanged (used to burn coins, which becomes a free amount when outdated)
> whitelist			Cross-chain off-chain asset whitelist address
> coinbaseaddress	Mining address for pledge mining (4)


#### hex created

```
0100000001a6bd2269b9ff68ec6ea9e1027d3977a0609892881c6113c8fd2a935ec2c89bf20000000000ffffffff030000000000000000516ac34c4df84b808094000000000000000000000000000000000000000a8080c0c0108080c0ebaa63713471656430346437326d6d676575767674747363377865663839767475743267397766376b6e383900407a10f35a00001976a914000000000000000000000000000000000000000a88ac00c817a8040000001976a91463a0562f00eed097ae0ada2ab48a2a26f52296ad88ac00000000
```




## 3. transaction signature


```
signrawtransaction "0100000001a6bd2269b9ff68ec6ea9e1027d3977a0609892881c6113c8fd2a935ec2c89bf20000000000ffffffff 030000000000000000516ac34c4df84b808094000000000000000000000000000000000000000a8080c0c0108080c0ebaa63713471656430346437326d6d676575767674747363377865663839767475743267397766376b6e383900407a10f35a00001976a914000000000000000000000000000000000000000a88ac00c817a8040000001976a91463a0562f00eed097ae0ada2ab48a2a26f52296ad88ac00000000"\[\{\" txid\"：\" a6bd2269b9ff68ec6ea9e1027d3977a0609892881c6113c8fd2a935ec2c89bf2\",\" vout\"：0,\" scriptpubkey\"：\"76a91463a0562f00eed097ae0ada2ab48a2a26f52296ad88ac\",\" amount\"：800\}\]\[\" KxnZH1ouGc3j1hESkajYUSwJGxTqfuXPpCT577pYopeaYxHjjKch\"\] wallet 
```



#### hex： after signature

```
0100000001a6bd2269b9ff68ec6ea9e1027d3977a0609892881c6113c8fd2a935ec2c89bf200000000644166fd69d4088d76ca44b58c72ed67af151344aa93765546d520fc88d2c174267cc53a415737fb0221138ae46812d6fcb22f92f475bfc30f390c698dc223904149412103656ffaa28a0cd36faccdb28dad7f72e33175c8984a3d1fb9310a6473ec2160a1ffffffff030000000000000000516ac34c4df84b808094000000000000000000000000000000000000000a8080c0c0108080c0ebaa63713471656430346437326d6d676575767674747363377865663839767475743267397766376b6e383900407a10f35a00001976a914000000000000000000000000000000000000000a88ac00c817a8040000001976a91463a0562f00eed097ae0ada2ab48a2a26f52296ad88ac00000000
```



## 4. enquiries on pledge
For ease of inquiry, you can use the getstateinfo interface to query specific lighthouse registration

Example:
```
root：~/ go/src/github.com/classzz/classzz#./czzctl getstateinfo
```
```
[
 {
 "exchange_id"：2,"
 "address"："cp36q430qrhdp9awptdz4dy29gn02g5k45ytdk9wcp","
 "toAddress_pk_hex"："0000000000000000000000000000000000000063","
 "staking_amount"：100000000000000,"
 "asset_flag"：16,"
 "fee"：0,"
 "keep_time"：0,"
 "white_list"：null,"
 "CoinBaseAddress"：["
 "cqurcmfxmz2xrp4wcx3776tvwl64rf7umvafq2r3qr""
 ]
 },
 {
 "exchange_id"：1,"
 "address"："cqurcmfxmz2xrp4wcx3776tvwl64rf7umvafq2r3qr","
 "toAddress_pk_hex"："000000000000000000000000000000000000000a","
 "staking_amount"：100000000000000,"
"asset_flag"：16,"
 "fee"：0,"
 "keep_time"：0,"
 "white_list"：null,"
 "CoinBaseAddress"：["
 "cp36q430qrhdp9awptdz4dy29gn02g5k45ytdk9wcp""
 ]
 }
]
```


#####  toAddress_pk_hex ： hexadecimal string representation for lighthouse address
