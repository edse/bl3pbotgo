/*
bl3p_1     | 2018/06/03 04:03:12 read: websocket: close 1006 (abnormal closure): unexpected EOF
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

const NormPrice = 100000.0
const NormAmount = 100000000.0

type Message struct {
	Date        int64
	Marketplace string
	Price_int   int64
	Type        string
	Amount_int  int64
}

func decodeMessage(message []byte) Message {
	var m Message
	err := json.Unmarshal([]byte(message), &m)
	if err != nil {
		log.Println("parse:", err)
	}
	log.Printf("decoded data: %s", m)
	return m
}

func storeMessage(data Message, pair string) {
	tags := map[string]string{"type": data.Type}
	fields := map[string]interface{}{
		"price":  float64(float64(data.Price_int) / NormPrice),
		"amount": float64(float64(data.Amount_int) / NormAmount),
	}

	Store(Database, pair, tags, fields, time.Unix(data.Date, 0))
}

func StartWebsocket(session Session, buy chan bool, sell chan bool) {
	fmt.Println("Starting websocket")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	c, _, err := websocket.DefaultDialer.Dial("wss://api.bl3p.eu/1/BTCEUR/trades", nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)

			// decode message
			data := decodeMessage(message)

			// save data on influx
			storeMessage(data, session.Pair)

			// analyse data
			Analyse(session, buy, sell)
		}
	}()

	for {
		select {
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			return
		}
	}
}
