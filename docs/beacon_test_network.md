## Documentation for building the lighthouse test network (Linux environment)


The executables compiled for each operating system have been placed on Github, eliminating the need to install the environment and making them more user-friendly for those who have no programming experience. The download method is specified in the documentation.

The main network of V3.2.0 beta version needs to download dogecoin, Litecoin, BTCCoin, BCHCoin and BSVCoin nodes and open RPC service, which is necessary, otherwise the main network will not operate normally

#### Example of Dogecoin node configuration:

```
server = 1
rpcuser =root # RPC user name
rpcpassword = admin # RPC password
rpcallowip =127.0.0.1 # Need to open all access 0.0.0.0.0/0
rpcbind = 0.0.0.0
rpcport = 9999 # RPC ports
txindex = 1
```
Command:
```
./dogecoind
```

---

#### Litecoin node configuration example:

```
server = 1
rpcuser =root # RPC user name
rpcpassword = admin # RPC password
rpcallowip =127.0.0.1 # Need to open all access 0.0.0.0.0/0
rpcbind = 0.0.0.0
rpcport = 19200 # RPC ports
txindex = 1
```

Command:

```
./litecoind
```

---

#### Example of BTCCoin node configuration:

```
server = 1
rpcuser =root # RPC user name
rpcpassword = admin # RPC password
rpcallowip =127.0.0.1 # Need to open all access 0.0.0.0.0/0
rpcbind = 0.0.0.0:19112
txindex = 1
rpcthreads = 50
```

Command:

```
./bitcoind
```

---

#### BchCoin Node Configuration Example:

```
server = 1
rpcuser =root # RPC user name
rpcpassword = admin # RPC password
rpcallowip =127.0.0.1 # Need to open all access 0.0.0.0.0/0
rpcbind = 0.0.0.0
port = 9113
txindex = 1
rpcthreads = 50
```

Command:

```
./bitcoind
```

----

#### BSVCoin Node Configuration Example:

```
server = 1
rpcuser =root # RPC user name
rpcpassword = admin # RPC password
rpcallowip =127.0.0.1 # Need to open all access 0.0.0.0.0/0
rpcbind = 0.0.0.0:19114
txindex = 1
rpcthreads = 50
excessiveblocksize = 2000000000
maxstackmemoryusageconsensus = 200000000
```

Command:

```
./bitcoind
```

---


## How do I install Classzz on Linux (Ubuntu)

#### 1.Prepare the Classzz environment
    Create the Classzz folder in the user root directory

#### 2.Download the Classzz program

    Dig and executable file download: https://github.com/classzz/classzz/releases/tag/v3.2.0-beta.1
    Download the purse executable file: https://github.com/classzz/czzwallet/releases/
    Download the native system executables and place them in the Classzz directory.

    Enter the url: https://raw.githubusercontent.com/classzz/classzz/master/csatable.zip
    Download the mining file csatable.bin in the same directory as the executable and unzip the zip file


#### 3.Configure the Classzz run file

**The following folder name, file name, case sensitive**

Create the Classzz. Conf file in the Classzz folder you just created


```
; Create the address of the data

datadir = data
testnet = 1


; RPC's whitelist policy (where the configuration is for everyone to connect to)

rpclisten = 0.0.0.0
whitelist = 0.0.0.0


; The user name and password for RPC

rpcuser = root
rpcpass = root


btccoinrpc = 127.0.0.1:9400
btccoinrpcuser = root
btccoinrpcpass = admin


Bchcoinrpc = 127.0.0.1:9401
Bchcoinrpcuser = root
Bchcoinrpcpass = admin


dogecoinrpc = 127.0.0.1:9402
dogecoinrpcuser = root
dogecoinrpcpass = admin



ltccoinrpc = 127.0.0.1:9403
ltccoinrpcuser = root
ltccoinrpcpass = admin



bsvcoinrpc = 127.0.0.1:9404
bsvcoinrpcuser = root
bsvcoinrpcpass = admin

```


Go to the user root directory

```
cd ~
```


Create the.czzctl folder and create the czzctl.conf file under the folder

```
rpcuser=root
rpcpass=root
```



Create the. czzwallet folder and under the folder create the czzwallet.conf file

```
username=root
password=root
```
Both of these are the user name and password for RPC





#### 4.Run Classzz


###### 1.Start the main network

Go to the Classzz directory and run the following command

```
./czzd -configfile="classzz.conf"
```

###### 2.Open a new command window and create the wallet in the new window

```
./czzwallet -create
```


###### 3. Launch the wallet

```

./czzwallet

```


###### 4.Close the main network restart command, if necessary be sure to use this command to restart

CTRL + C

Then do the first step


## Address creation and WIF private key of the rest of the main network converted to CZZ address private key and address

###### address creation

```
./createaddress
```

###### The WIF private key of the rest of the main network is converted to the private key and address of CZZ address

```
./convertAddress -p="external key" -t=""
```
