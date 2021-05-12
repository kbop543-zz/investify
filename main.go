package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type StockQuote struct {
	Symbol           string
	CompanyName      string
	PrimaryExchange  string
	IexRealtimePrice float32
	Change           float32
	ChangePercent    float32
	MarketCap        int64
	Volume           int64
	PeRatio          float32
}

type StockIncome struct {
	Symbol      string
	CompanyName string
	EBITLastQ   float32
	EBITSecondQ float32
	PeRatio     float32
}

type StockIncomeMap struct {
	Symbol string
	Income []Income
}

type Income struct {
	Ebit          float32
	FiscalQuarter int64
}

const token = "pk_d88e97f417f143f69dd97fccd5bb662b"
const quoteUrl = "https://cloud.iexapis.com/stable/stock/%s/quote?token=%s"
const incomeUrl = "https://cloud.iexapis.com/stable/stock/%s/income?period=%s&last=%s&token=%s"

func main() {
	//techStocks := []string{"aapl", "msft", "fb", "tsla", "shop", "googl", "amzn", "nflx"}
	weedStocks := []string{"cgc", "apha", "acb", "cron"}

	//fmt.Println("Best Tech Stock:", findBestStock(techStocks).Symbol)
	fmt.Println("Best Weed Stock:", findBestStock(weedStocks).Symbol)

}

func findBestStock(symbols []string) StockIncome {
	/*Find increasing ebit for 2 quarters, use highest PE ratio as tie breaker
	1. Use one flow to find basic stock quote info
	2. Use another flow to find income stock quote info aka ebit info for last two quarters
	3. Step for 1 and 2 to be completed and calulate which stock to choose using the data
	*/

	stockQuotes := make(chan StockQuote)
	stockIncomes := make(chan StockIncome)

	defer close(stockQuotes)

	go getSymbols(symbols, stockQuotes)                            // Step 1
	go getIncomeForQuotes(len(symbols), stockQuotes, stockIncomes) // Step 2

	bestStock := <-stockIncomes
	fmt.Println("coming from channel", bestStock.Symbol)

	for i := 0; i < len(symbols)-1; i++ {
		stock := <-stockIncomes
		fmt.Println("coming from channel", stock.Symbol)

		if stock.EBITLastQ > stock.EBITSecondQ {
			if stock.EBITLastQ > bestStock.EBITLastQ && stock.PeRatio > bestStock.PeRatio {
				bestStock = stock
			}
		} else {
			if !(bestStock.EBITLastQ > bestStock.EBITSecondQ) {
				if stock.EBITLastQ > bestStock.EBITLastQ && stock.PeRatio > bestStock.PeRatio {
					bestStock = stock
				}
			}
		}
	}

	return bestStock

}

func getSymbols(symbols []string, stockQuotes chan StockQuote) {

	for _, symbol := range symbols {

		go func(sym string, stockQuotes chan StockQuote) {
			getSymbol(sym, stockQuotes)
		}(symbol, stockQuotes)
	}

}

func getIncomeForQuotes(count int, stockQuotes chan StockQuote, stockIncomes chan StockIncome) {

	for i := 0; i < count; i++ {
		str := <-stockQuotes
		myClient := &http.Client{}

		r, err := myClient.Get(fmt.Sprintf(incomeUrl, str.Symbol, "quarter", "2", token))

		if err != nil {
			log.Fatalf("Error getting symbol quote: %s", err)
		}
		defer r.Body.Close()

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatalln(err)
		}

		var stockIncomeMap StockIncomeMap
		fmt.Println("body", string(body))
		err = json.Unmarshal(body, &stockIncomeMap)
		if err != nil {
			fmt.Println("error:", err)
		}

		var stockIncome StockIncome
		stockIncome.Symbol = stockIncomeMap.Symbol
		stockIncome.CompanyName = str.CompanyName

		if len(stockIncomeMap.Income) == 1 {
			stockIncome.EBITLastQ = stockIncomeMap.Income[0].Ebit
		}

		if len(stockIncomeMap.Income) == 2 {
			stockIncome.EBITLastQ = stockIncomeMap.Income[1].Ebit
		}

		stockIncome.PeRatio = str.PeRatio

		stockIncomes <- stockIncome
	}
}

func getSymbol(str string, stockQuotes chan StockQuote) {
	myClient := &http.Client{}

	r, err := myClient.Get(fmt.Sprintf(quoteUrl, str, token))

	if err != nil {
		log.Fatalf("Error getting symbol quote: %s", err)
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var stockQuote StockQuote
	err = json.Unmarshal(body, &stockQuote)
	if err != nil {
		fmt.Println("error:", err)
		fmt.Println("symbol:", str)
		fmt.Println("body:", string(body))
	}

	stockQuotes <- stockQuote
}
