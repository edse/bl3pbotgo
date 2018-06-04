package main

import (
	"fmt"
	"time"
)

type Session struct {
	Status               string    // session status
	Ma1                  uint      // price moving average low
	Ma2                  uint      // price moving average high
	Ma3                  uint      // diff moving average
	DataRange            string    // data range to be analysed
	DataGroup            string    // group data by time interval
	DataInterval         string    // interval between analyses
	CryptoBalanceAtStart float64   // balance in crypto at session start
	EuroBalanceAtStart   float64   // balance in euros at session start
	CryptoBalance        float64   // crypto balance
	EuroBalance          float64   // euro balance
	Pair                 string    // session trading pair
	StartedAt            time.Time // time when session started
	EndedAt              time.Time // time when session ended
}

func main() {
	fmt.Println(intro)
	buySignal := make(chan bool)
	sellSignal := make(chan bool)

	// TODO: add bellow settings as parameters
	session := Session{
		Status:       "started",
		Ma1:          12,
		Ma2:          26,
		Ma3:          9,
		DataRange:    "24h",
		DataGroup:    "15m",
		DataInterval: "15",
		Pair:         "BTCEUR",
		StartedAt:    time.Now(),
	}

	StartWebsocket(session, buySignal, sellSignal)

	select {
	case <-buySignal:
		fmt.Println("buy signal received")
		Buy()
	case <-sellSignal:
		fmt.Println("sell signal received")
		Sell()
	default:
		fmt.Println("    .")
		time.Sleep(50 * time.Millisecond)
	}
}

const intro = `
██████╗ ██╗     ██████╗ ██████╗     ██████╗  ██████╗ ████████╗
██╔══██╗██║     ╚════██╗██╔══██╗    ██╔══██╗██╔═══██╗╚══██╔══╝
██████╔╝██║      █████╔╝██████╔╝    ██████╔╝██║   ██║   ██║   
██╔══██╗██║      ╚═══██╗██╔═══╝     ██╔══██╗██║   ██║   ██║   
██████╔╝███████╗██████╔╝██║         ██████╔╝╚██████╔╝   ██║   
╚═════╝ ╚══════╝╚═════╝ ╚═╝         ╚═════╝  ╚═════╝    ╚═╝   
`
