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
	CounterDays [NumberDays]HtlcCounter `json:"counter_days"`
	Position    int                     `json:"position"`
	LastUpdate  time.Time               `json:"last_update"`
	Day         int                     `json:"current_day"`
}
type HtlcEvent int

const (
	Settled HtlcEvent = iota
	LinkFail
	ForwardFail
	ForwardEvent
)

const NumberDays = 30

func (stat *HtlcStatistics) count(event HtlcEvent) {

	prevDay := stat.Day
	stat.Day = time.Now().Minute()
	if prevDay != stat.Day {
		stat.Position = (stat.Position + 1) % NumberDays
		stat.CounterDays[stat.Position].Settled = 0
		stat.CounterDays[stat.Position].LinkFails = 0
		stat.CounterDays[stat.Position].ForwardEvents = 0
		stat.CounterDays[stat.Position].ForwardFails = 0
	}
	stat.LastUpdate = time.Now()

	switch event {
	case Settled:
		stat.CounterDays[stat.Position].Settled += 1
	case LinkFail:
		stat.CounterDays[stat.Position].LinkFails += 1
	case ForwardFail:
		stat.CounterDays[stat.Position].ForwardFails += 1
	case ForwardEvent:
		stat.CounterDays[stat.Position].ForwardEvents += 1
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
	return fmt.Sprintf("HTLC-Stats (Last 1 Days): Settled %d, LinkFail %d, ForwardFail %d, ForwardEvents %d", stat.CounterDays[stat.Day].Settled,
		stat.CounterDays[stat.Day].LinkFails, stat.CounterDays[stat.Day].ForwardFails, stat.CounterDays[stat.Day].ForwardEvents)
}

func (stat *HtlcStatistics) get7Days() string {
	var totalWeek HtlcCounter
	for i := 0; i < 7; i++ {
		day := (stat.Position - i + NumberDays) % NumberDays
		totalWeek.Settled += stat.CounterDays[day].Settled
		totalWeek.ForwardEvents += stat.CounterDays[day].ForwardEvents
		totalWeek.ForwardFails += stat.CounterDays[day].ForwardFails
		totalWeek.LinkFails += stat.CounterDays[day].LinkFails
	}
	return fmt.Sprintf("HTLC-Stats (Last 7 Days): Settled %d, LinkFail %d, ForwardFail %d, ForwardEvents %d", totalWeek.Settled,
		totalWeek.LinkFails, totalWeek.ForwardFails, totalWeek.ForwardEvents)

}

func (stat *HtlcStatistics) get30Days() string {
	var totalMonth HtlcCounter
	for i := 0; i < 30; i++ {
		day := (stat.Position - i + NumberDays) % NumberDays
		totalMonth.Settled += stat.CounterDays[day].Settled
		totalMonth.ForwardEvents += stat.CounterDays[day].ForwardEvents
		totalMonth.ForwardFails += stat.CounterDays[day].ForwardFails
		totalMonth.LinkFails += stat.CounterDays[day].LinkFails
	}
	return fmt.Sprintf("HTLC-Stats (Last 30 Days): Settled %d, LinkFail %d, ForwardFail %d, ForwardEvents %d", totalMonth.Settled,
		totalMonth.LinkFails, totalMonth.ForwardFails, totalMonth.ForwardEvents)

}

func (stat *HtlcStatistics) postStats(ctx context.Context) {
	var cycleTime time.Duration = 6 * time.Second

	timer := time.NewTimer(cycleTime)
	defer timer.Stop()
	var wg sync.WaitGroup

	for {
		wg.Add(1)
		go func() {
			<-timer.C
			log.Debug("Posting HTLC Statistics via Telegram")
			message := stat.get1Day() + "\n" + stat.get7Days() + "\n" + stat.get30Days()
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
