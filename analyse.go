package main

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	Database   = "bl3pbot"
	DiffLabel  = "BTCEUR_MA1_MA2_DIFF"
	MacdLabel  = "BTCEUR_MACD"
	TrendLabel = "BTCEUR_TREND"
)

type Current struct {
	Date  time.Time
	Price float64
	Ma1   float64
	Ma2   float64
}

type MasDiff struct {
	Date time.Time
	Ma1  float64
	Ma2  float64
	Diff float64
}

type Macd struct {
	Date   time.Time
	Signal float64
	Macd   float64
	Hist   float64
}

type Trend struct {
	Date  time.Time
	Trend int64
	State string
	H1    float64
	H2    float64
}

func storeTrend(data Trend) {
	fmt.Println("Storing TREND data: ", data)
	tags := map[string]string{"type": "TREND"}
	fields := map[string]interface{}{
		"trend": data.Trend,
		"state": data.State,
		"h1":    data.H1,
		"h2":    data.H2,
	}

	Store(Database, TrendLabel, tags, fields, data.Date)
}

func storeDiff(data MasDiff) {
	fmt.Println("Storing MAs diff data: ", data)
	tags := map[string]string{"type": "MAs diff"}
	fields := map[string]interface{}{
		"ma1":  data.Ma1,
		"ma2":  data.Ma2,
		"diff": data.Diff,
	}

	Store(Database, DiffLabel, tags, fields, data.Date)
}

func storeMacd(data Macd) {
	tags := map[string]string{"type": "MACD"}
	fields := map[string]interface{}{
		"signal": data.Signal,
		"macd":   data.Macd,
		"hist":   data.Hist,
	}

	Store(Database, MacdLabel, tags, fields, data.Date)
}

func checkTrend(session Session, current Current, buy chan bool, sell chan bool) {
	/*
		Check the last 2 records from the last 30m grouped by 1m

		trend:
			int(-10): when the trending is down and a sell action is required
			int(-1): when the trending is down
			int(0): when in no trend or no enough data
			int(1): when the trending is up
			int(10): when the trending is up and a sell action is required
	*/
	fmt.Println("Checking trend")

	var _macds [2]float64
	var _signals [2]float64

	c := GetClient()
	defer c.Close()

	// SIGNAL
	query := fmt.Sprintf(
		"SELECT moving_average(mean(\"diff\"), %v) as ma3, mean(\"diff\") as diff FROM \"%s\" WHERE time > now() - %s GROUP BY time(%s) fill(linear)",
		session.Ma3,
		DiffLabel,
		session.DataRange,
		session.DataGroup,
	)
	res := Query(c, Database, query)
	if len(res[0].Series) > 0 {
		fmt.Println("MAs diff data len: ", len(res[0].Series[0].Values))

		row_a := res[0].Series[0].Values[len(res[0].Series[0].Values)-2]
		_signals[0], _ = row_a[1].(json.Number).Float64()
		_macds[0], _ = row_a[2].(json.Number).Float64()

		row_b := res[0].Series[0].Values[len(res[0].Series[0].Values)-1]
		_signals[1], _ = row_b[1].(json.Number).Float64()
		_macds[1], _ = row_b[2].(json.Number).Float64()

		fmt.Println("signals: ", _signals)
		fmt.Println("macds: ", _macds)

		macdData := Macd{
			Date:   current.Date,
			Signal: _signals[1],
			Macd:   _macds[1],
			Hist:   _macds[1] - _signals[1],
		}
		fmt.Println("MACD data: ", macdData)

		// save macd data to influx
		storeMacd(macdData)
	}

	// HIST
	query = fmt.Sprintf(
		"SELECT mean(\"hist\") as hist FROM \"%s\" WHERE time > now() - %s GROUP BY time(%s) fill(linear)",
		MacdLabel,
		session.DataRange,
		session.DataGroup,
	)
	res = Query(c, Database, query)
	if len(res[0].Series) > 0 {
		fmt.Println("HIST data len: ", len(res[0].Series[0].Values))

		var trend int64
		var state string

		r := res[0].Series[0].Values[len(res[0].Series[0].Values)-2]
		h0, _ := r[1].(json.Number).Float64()

		r = res[0].Series[0].Values[len(res[0].Series[0].Values)-1]
		h1, _ := r[1].(json.Number).Float64()

		fmt.Println("H1: ", h0)
		fmt.Println("H2: ", h1)

		if h0 < h1 {
			// up trend
			if h0 <= 0 && h1 >= 0 {
				// buy
				trend = 10
				state = "buy"
				fmt.Println("STATE (buy): ", state)
				Buy()
			} else {
				trend = 1
				state = "up"
				fmt.Println("STATE (up): ", state)
			}
		}

		if h0 > h1 {
			// down trend
			if h1 <= 0 && h0 >= 0 {
				// sell
				trend = -10
				state = "sell"
				fmt.Println("STATE (sell): ", state)
				Sell()
			} else {
				trend = -1
				state = "down"
				fmt.Println("STATE (down): ", state)
			}
		}

		if h0 == h1 {
			// no trend
			trend = 0
			state = "sideways"
			fmt.Println("STATE (side): ", state)
		}

		trendData := Trend{
			Date:  current.Date,
			Trend: trend,
			State: state,
			H1:    h0,
			H2:    h1,
		}
		fmt.Println("TREND data: ", trendData)

		// save trend data to influx
		storeTrend(trendData)
	}
}

func Analyse(session Session, buy chan bool, sell chan bool) {
	fmt.Println("Analysing influx data")

	current := Current{}

	c := GetClient()
	defer c.Close()

	// PRICE
	query := fmt.Sprintf(
		"SELECT mean(\"price\") as price FROM \"%s\" WHERE time > now() - %s GROUP BY time(%s) fill(previous)",
		session.Pair,
		session.DataRange,
		session.DataGroup,
	)
	res := Query(c, Database, query)
	if len(res[0].Series) > 0 {
		fmt.Println("price data len: ", len(res[0].Series[0].Values))

		row := res[0].Series[0].Values[len(res[0].Series[0].Values)-1]
		fmt.Println("price last data: ", row)

		t, _ := time.Parse(time.RFC3339, row[0].(string))
		p, _ := row[1].(json.Number).Float64()

		fmt.Println("last time: ", t)
		fmt.Println("last price: ", p)

		current.Date = t
		current.Price = p
	}

	// MA1
	query = fmt.Sprintf(
		"SELECT moving_average(mean(\"price\"), %v) as ma1 FROM \"%s\" WHERE time > now() - %s GROUP BY time(%s) fill(linear)",
		session.Ma1,
		session.Pair,
		session.DataRange,
		session.DataGroup,
	)
	res = Query(c, Database, query)

	if len(res[0].Series) > 0 {
		fmt.Println("ma12 data len: ", len(res[0].Series[0].Values))
		row := res[0].Series[0].Values[len(res[0].Series[0].Values)-1]
		fmt.Println("ma12 last data: ", row)

		ma1, _ := row[1].(json.Number).Float64()

		fmt.Println("last ma12: ", ma1)
		current.Ma1 = ma1
	}

	// MA2
	query = fmt.Sprintf(
		"SELECT moving_average(mean(\"price\"), %v) as ma2 FROM \"%s\" WHERE time > now() - %s GROUP BY time(%s) fill(linear)",
		session.Ma2,
		session.Pair,
		session.DataRange,
		session.DataGroup,
	)
	res = Query(c, Database, query)

	if len(res[0].Series) > 0 {
		fmt.Println("ma26 data len: ", len(res[0].Series[0].Values))

		if len(res[0].Series[0].Values) > 0 {
			row := res[0].Series[0].Values[len(res[0].Series[0].Values)-1]
			fmt.Println("ma26 last data: ", row)

			ma2, _ := row[1].(json.Number).Float64()

			fmt.Println("last ma26: ", ma2)
			current.Ma2 = ma2
		}
	}

	// MA DIFF
	if current.Ma1 > 0 && current.Ma2 > 0 {
		data := MasDiff{
			Date: current.Date,
			Ma1:  current.Ma1,
			Ma2:  current.Ma2,
			Diff: current.Ma1 - current.Ma2,
		}
		fmt.Println("MA DIFF data: ", data)

		// save diff data to influx
		storeDiff(data)
	}

	checkTrend(session, current, buy, sell)
}
