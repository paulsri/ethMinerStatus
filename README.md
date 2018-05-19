# ethMinerStatus

Usage: ethMinerStatus dashboard || telegram

* dashboard: a web application is run as described below
* telegram: a report is generated hourly and sent to your chat of choice

Simple script that queries the Ethereum Go client via JSON RPC for attributes such as

* coinbase
* peerCount
* blockNumber

and displays the response in a table.

A summary table is also printed above the table containing:

* Balance of WTCT
* Total Miners
* Histogram of Block Numbers
* Histogram of Peers

User-specific parameters can be provided via a configuration file named `config.json`
