package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "io/ioutil"
    "log"
    "strconv"
    "time"
    "math/big"
    "strings"
    "os"
    "net/url"
    "sort"
    "github.com/spf13/viper"
)

func showDashboard(w http.ResponseWriter, r *http.Request){

    primeThePump()
    balance, data := retrieveAllData()
    reportBlockNumber, blockError, report, minerCount, zeroBlockMiners := analyzeData(data)
    fmt.Println("blockError: ", blockError)

    fmt.Fprintf(w,"<!DOCTYPE html> <html> <head> <title>Miner Dashboard</title> </head> <body>")

    fmt.Fprintln(w, "Miner Dashboard")

    fmt.Fprintln(w,"<br>")
    fmt.Fprintln(w,"<br>")

    fmt.Fprintln(w, "Balance: " + balance + " WTCT")
    fmt.Fprintln(w,"<br>")
    fmt.Fprintln(w, "Total Miners: " + strconv.Itoa(minerCount))

    fmt.Fprintln(w,"<br>")
    fmt.Fprintln(w,"<br>")

    // blocks
    fmt.Fprintln(w, "Section: Block:")
    fmt.Fprintln(w,"<br>")
    for keyBlock, valueBlock := range reportBlockNumber {
      fmt.Fprintln(w, keyBlock + ": " + strconv.Itoa(valueBlock))
      fmt.Fprintln(w,"<br>")
    }
    fmt.Fprintln(w, zeroBlockMiners)

    fmt.Fprintln(w,"<br>")

    // sort map by keys
    var keys []string
    for k := range report {
      keys = append(keys, k)
    }
    sort.Strings(keys)

    // peers
    fmt.Fprintln(w, "Section: Peers:")
    fmt.Fprintln(w,"<br>")
    for _, k := range keys {
      fmt.Fprintln(w, k + ": " + strconv.Itoa(report[k]))
      fmt.Fprintln(w,"<br>")
    }

    fmt.Fprintln(w,"<br>")
    fmt.Fprintln(w,"<br>")

    fmt.Fprintln(w, "<table>")
    fmt.Fprintln(w, "<tr><th>#</th><th>Miner</th><th>Block</th><th>Peers</th><th>Wallet</th></tr>")

    miners := viper.GetStringSlice("miners")

    for i := 0; i < len(miners); i++ {
      fmt.Fprintln(w, "<tr>")
      fmt.Fprintf(w, "<td>" + strconv.Itoa(i+1) + "</td>")
      fmt.Fprintf(w, "<td>" + miners[i] + "</td>")
      fmt.Fprintln(w, "<td>" + data[i][0] + "</td>")
      fmt.Fprintln(w, "<td>" + data[i][1] + "</td>")
      fmt.Fprintln(w, "<td>" + data[i][2] + "</td>")
      fmt.Fprintln(w, "</tr>")
    }

    fmt.Fprintln(w, "</table>")
    fmt.Fprintf(w,"</body></html>")
}

func retrieveAllData () (string, [][]string) {
    miners := viper.GetStringSlice("miners")
    balance := queryMiner(miners[0], true, 300)
    currentBalance := extractResult(balance[0])

    miner2DData := [][]string{}

    for i := 0; i < len(miners); i++ {
      singleMinerResponse := queryMiner(miners[i], false, 200)

      currentBlock := extractResult(singleMinerResponse[0])
      currentPeers := extractResult(singleMinerResponse[1])
      currentWallet := extractResult(singleMinerResponse[2])

      miner2DData = append(miner2DData, []string{miners[i], hexToInt(currentBlock), hexToInt(currentPeers), currentWallet})
    }

    return bigHexToInt(currentBalance), miner2DData
}

func hexToInt (hex string) string {
    convertedHex, err := strconv.ParseInt(strings.TrimLeft(hex, "0x"), 16, 64)
    if err != nil {
      log.Print(err)
    }
    return strconv.FormatInt(convertedHex, 10)
}

func bigHexToInt (hex string) string {
    bi := big.NewInt(0)
    readableBalance := big.NewFloat(0)
    if _, ok := bi.SetString(strings.TrimLeft(hex, "0x"), 16); ok {
      denominator := big.NewFloat(1000000000000000000)
      numerator := new(big.Float).SetInt(bi)
      readableBalance := new(big.Float).Quo(numerator, denominator)
      return readableBalance.String()
    }
    return readableBalance.String()
}

func extractResult(response string) string {
    type ethResponse struct {
      JSONRPC string `json:"jsonrpc"`
      ID      int    `json:"id"`
      Result  string `json:"result"`
    }

    minerData := []byte(response)
    var minerResponse ethResponse
    err := json.Unmarshal(minerData, &minerResponse)
    if err != nil {
      log.Print(err)
    }
    return minerResponse.Result
}

func queryMiner(miner string, balance bool, timeout int) []string {

    type ethRequest struct {
      JSONRPC string        `json:"jsonrpc"`
    	ID      int           `json:"id"`
    	Method  string        `json:"method"`
      Params  []interface{} `json:"params"`
    }

    if balance {

      params := []interface{}{viper.GetString("wallet"), "latest"}

      balanceJson, err := json.Marshal(ethRequest{Method:"eth_getBalance",ID:1,Params:params})

      if err != nil {
        log.Print(err)
      }

      balanceResponse := makeRequest(miner, balanceJson, timeout)

      responses := []string{balanceResponse}

      return responses
    } else {

      blockNumberJson, err := json.Marshal(ethRequest{Method:"eth_blockNumber",ID:83})
      peerCountJson, err := json.Marshal(ethRequest{Method:"net_peerCount",ID:74})
      coinbaseJson, err := json.Marshal(ethRequest{Method:"eth_coinbase",ID:64})

      if err != nil {
        log.Print(err)
      }

      blockNumberResponse := makeRequest(miner, blockNumberJson, timeout)
      peerCountResponse := makeRequest(miner, peerCountJson, timeout)
      coinbaseResponse := makeRequest(miner, coinbaseJson, timeout)

      responses := []string{blockNumberResponse, peerCountResponse, coinbaseResponse}

      return responses
    }
}

func makeRequest(miner string, jsonMethod []byte, timeoutMS int) string {
    req, err := http.NewRequest("POST", "http://" + miner + ".local:8545", bytes.NewBuffer(jsonMethod))
    req.Header.Set("Content-Type", "application/json")


    timeout := time.Duration(time.Duration(timeoutMS) * time.Millisecond)

    client := &http.Client{Timeout: timeout}
    resp, err := client.Do(req)
    if err != nil {
      log.Print(err)
      return "0"
    }
    body, err := ioutil.ReadAll(resp.Body)

    fmt.Print("Response: ", string(body))
    resp.Body.Close()
    return string(body)
}

func analyzeData(data [][]string) (map[string]int, bool, map[string]int, int, string) {
    var reportBlockNumber map[string]int
    reportBlockNumber = make(map[string]int)
    var report map[string]int
    report = make(map[string]int)

    zeroBlockMiners := "Zero Block Miners: "

    report["Peers Bucket A: 0"] = 0
    report["Peers Bucket B: 1"] = 0
    report["Peers Bucket C: 2-5"] = 0
    report["Peers Bucket D: 6-10"] = 0
    report["Peers Bucket E: 11-15"] = 0
    report["Peers Bucket F: 16-20"] = 0
    report["Peers Bucket G: 20+"] = 0

    for i := 0; i < len(data); i++ {
      // block number
      reportBlockNumber["Block Number: " + data[i][1]]++

      // keep track of miners mining 0 blocks
      if data[i][1] == strconv.Itoa(0) {
        zeroBlockMiners = zeroBlockMiners + data[i][0] + " "
      }

      // peers
      peers, err := strconv.Atoi(data[i][2])

      if err != nil {
        log.Print(err)
      }

      if peers == 0 {
        report["Peers Bucket A: 0"]++
      } else if peers == 1 {
        report["Peers Bucket B: 1"]++
      } else if peers > 1 && peers < 6 {
        report["Peers Bucket C: 2-5"]++
      } else if peers > 5 && peers < 11 {
        report["Peers Bucket D: 6-10"]++
      } else if peers > 10 && peers < 16 {
        report["Peers Bucket E: 11-15"]++
      } else if peers > 15 && peers < 21 {
        report["Peers Bucket F: 16-20"]++
      } else if peers > 20 {
        report["Peers Bucket G: 20+"]++
      }
    }

    // reportBlockNumber error
    var blockErrorState bool = false
    numberBlockErrors := len(reportBlockNumber)
    // fmt.Println("reportBlockNumber Count: ", numberBlockErrors)
    if numberBlockErrors > 1 {
      blockErrorState = true
      fmt.Printf("Block Histogram Error Identified!: %d Different Blocks Being Mined!\n", numberBlockErrors)
    }
    // fmt.Println("blockErrorState: ", blockErrorState)

    // miner count
    minerCount := 0
    for _, valuePeers := range report {
      minerCount = minerCount + valuePeers
    }

    return reportBlockNumber, blockErrorState, report, minerCount, zeroBlockMiners
}

func primeThePump() {
    miners := viper.GetStringSlice("miners")
    for i := 0; i < len(miners); i++ {
      balance := queryMiner(miners[i], true, 25)
      balance[0] = balance[0]
    }
    time.Sleep(5 * time.Second)
}

func server() {
    http.HandleFunc("/", showDashboard)
    http.ListenAndServe(":" + viper.GetString("port"), nil)
}

func telegram() {
  for {
    primeThePump()
    balance, data := retrieveAllData()
    reportBlockNumber, blockError, report, minerCount, zeroBlockMiners := analyzeData(data)

    newLine := "%0A"
    doubleNewLine := "%0A%0A"

    fmt.Println("Balance: ", balance)
    fmt.Printf("Report: %v\n", report)

    tgURL          := "https://api.telegram.org/bot"
    tgAPIKey       := viper.GetString("telegramAPIKey")
    tgEndpoint     := "/sendMessage?chat_id="
    tgChatID       := viper.GetString("telegramChatID")
    tgParameter    := "&text="
    tgReportHeader := url.QueryEscape("ethMinerStatus Hourly Report") + newLine
    tgReport       := ""

    // header: block bucket number
    tgReport = tgReport + newLine + url.QueryEscape("Section: Block Number")
    for keyBlock, valueBlock := range reportBlockNumber {
      tgReport = tgReport + newLine + keyBlock + url.QueryEscape(": ") + strconv.Itoa(valueBlock)
    }
    tgReport = tgReport + newLine + zeroBlockMiners

    // sort map by keys
    var keys []string
    for k := range report {
      keys = append(keys, k)
    }
    sort.Strings(keys)

    // header: peers
    tgReport = tgReport + doubleNewLine + url.QueryEscape("Section: Peers")
    for _, k := range keys {
      tgReport = tgReport + newLine + url.QueryEscape(k) + url.QueryEscape(": ") + strconv.Itoa(report[k])
    }

    // header: summary
    tgReport = tgReport + doubleNewLine + url.QueryEscape("Section: Summary")
    summaryBalance := "Balance: " + balance + " WTCT"
    summaryTotal   := "Total Miners: " + strconv.Itoa(minerCount)
    tgReport = tgReport + newLine + url.QueryEscape(summaryBalance) + newLine + url.QueryEscape(summaryTotal)

    // final telegram request
    tgRequest := tgURL + tgAPIKey + tgEndpoint + tgChatID + tgParameter + tgReportHeader + tgReport

    if blockError == true {
      _, err := http.Get(tgRequest)
      if err != nil {
        log.Print(err)
      }
    } else {
      tgRequest := tgURL + tgAPIKey + tgEndpoint + tgChatID + tgParameter + "Everything's Fine"
      _, err := http.Get(tgRequest)
      if err != nil {
        log.Print(err)
      }
    }

    time.Sleep(3600 * time.Second)
  }
}

func main() {

  viper.SetConfigName("config")
  viper.AddConfigPath(".")
  err := viper.ReadInConfig()
  if err != nil {
    panic(fmt.Errorf("Fatal error config file: %s \n", err))
  }
  fmt.Printf("Using config: %s\n", viper.ConfigFileUsed())

  if len(os.Args) == 2 {
    if os.Args[1] == "dashboard" {
      fmt.Println("ethMinerStatus: Dashboard Mode!")
      server()
    } else if os.Args[1] == "telegram" {
      fmt.Println("ethMinerStatus: Telegram Mode!")
      telegram()
    }
  }
  fmt.Println("ethMinerStatus: Usage: ethMinerStatus dashboard || telegram")
}
