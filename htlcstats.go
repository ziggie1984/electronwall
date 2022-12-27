package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type HtlcCounter struct {
	LinkFails     int `json:"linkfails"`
	ForwardFails  int `json:"forwardsfails"`
	Settled       int `json:"settled"`
	ForwardEvents int `json:"forwardevents"`
}

type HtlcStatistics struct {
	CounterWeek [Last7Days]HtlcCounter `json:"week_summary"`
	LastUpdate  time.Time              `json:"last_update"`
	WeekDay     int                    `json:"current_day"`
}
type HtlcEvent int

const (
	Settled HtlcEvent = iota
	LinkFail
	ForwardFail
	ForwardEvent
)

const Last7Days = 7

func (stat *HtlcStatistics) count(event HtlcEvent) {

	prevDay := stat.WeekDay
	stat.WeekDay = time.Now().Day() % Last7Days
	if prevDay != stat.WeekDay {
		stat.CounterWeek[stat.WeekDay].Settled = 0
		stat.CounterWeek[stat.WeekDay].LinkFails = 0
		stat.CounterWeek[stat.WeekDay].ForwardEvents = 0
		stat.CounterWeek[stat.WeekDay].ForwardFails = 0
	}
	stat.LastUpdate = time.Now()

	switch event {
	case Settled:
		stat.CounterWeek[stat.WeekDay].Settled += 1
	case LinkFail:
		stat.CounterWeek[stat.WeekDay].LinkFails += 1
	case ForwardFail:
		stat.CounterWeek[stat.WeekDay].ForwardFails += 1
	case ForwardEvent:
		stat.CounterWeek[stat.WeekDay].ForwardEvents += 1
	}
}

func NewHtlcStats(ctx context.Context) *HtlcStatistics {
	htlcState := &HtlcStatistics{}
	err := htlcState.loadState()
	if err != nil {
		htlcState.LastUpdate = time.Now()
	}
	return htlcState
}

func (stat *HtlcStatistics) get1Day() string {
	return fmt.Sprintf("HTLC-Stats (1 Day): Settled %d, LinkFail %d, ForwardFail %d, ForwardEvents %d", stat.CounterWeek[stat.WeekDay].Settled,
		stat.CounterWeek[stat.WeekDay].LinkFails, stat.CounterWeek[stat.WeekDay].ForwardFails, stat.CounterWeek[stat.WeekDay].ForwardEvents)
}

func (stat *HtlcStatistics) get7Days() string {
	var totalWeek HtlcCounter
	for _, dayStat := range stat.CounterWeek {
		totalWeek.Settled += dayStat.Settled
		totalWeek.ForwardEvents += dayStat.ForwardEvents
		totalWeek.ForwardFails += dayStat.ForwardFails
		totalWeek.LinkFails += dayStat.LinkFails
	}
	return fmt.Sprintf("HTLC-Stats (7 Days): Settled %d, LinkFail %d, ForwardFail %d, ForwardEvents %d", totalWeek.Settled,
		totalWeek.LinkFails, totalWeek.ForwardFails, totalWeek.ForwardEvents)

}

func (stat *HtlcStatistics) postStats(ctx context.Context) {
	var cycleTime time.Duration = 60 * time.Minute

	timer := time.NewTimer(cycleTime)
	defer timer.Stop()
	var wg sync.WaitGroup

	for {
		wg.Add(1)
		go func() {
			<-timer.C
			log.Debug("Posting HTLC Statistics")
			message := stat.get1Day() + "\n" + stat.get7Days()
			err := telegramNotifier.Notify(message)
			if err != nil {
				log.Error("htlc stats error", err)
			}
			stat.saveState()
			timer.Reset(cycleTime)
			wg.Done()
		}()
		wg.Wait()

	}
}

func (stat *HtlcStatistics) saveState() error {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Error(err)
		return err
	}

	// Construct the path to the file in the home directory
	fileName := homeDir + "/htlcstats.json"

	os.Stat(fileName)
	f, ferr := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0666)
	if ferr != nil {
		log.Errorf("Error saving htlc stats to %s: %s", fileName, ferr)
		return ferr
	}
	defer f.Close()
	content, _ := json.MarshalIndent(*stat, "", " ")
	_, err = f.Write(content)

	if err != nil {
		log.Errorf("Error writing htlc stats to %s: %s", fileName, err)
		return err
	}

	return nil

}

func (stat *HtlcStatistics) loadState() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Error(err)
		return err
	}

	// Construct the path to the file in the home directory
	fileName := homeDir + "/htlcstats.json"
	bytes, err := os.ReadFile(fileName)
	if err != nil {
		log.Errorf("Error reading htlc stats from file %s,%s", fileName, err)
		return err
	}
	err = json.Unmarshal(bytes, &stat)
	if err != nil {
		log.Errorf("Error unmarshal file %s,%s", fileName, err)
		return err
	}

	return nil
}

func (stat *HtlcStatistics) DispatchHtlcStats(ctx context.Context) {

	// Htlc statistics
	go func() {
		stat.postStats(ctx)

		// release wait group for channel acceptor
		ctx.Value(ctxKeyWaitGroup).(*sync.WaitGroup).Done()
	}()
}
