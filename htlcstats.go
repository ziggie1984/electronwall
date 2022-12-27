package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type HtlcCounter struct {
	LinkFails     int
	ForwardFails  int
	Settled       int
	ForwardEvents int
}

type HtlcStatistics struct {
	CounterWeek [Last7Days]HtlcCounter
	LastUpdate  time.Time
	WeekDay     int
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

	stat.WeekDay = time.Now().Day() % Last7Days
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

	return &HtlcStatistics{
		LastUpdate: time.Now(),
	}
}

func (stat *HtlcStatistics) postStats(ctx context.Context, telegramNotifier *TelegramNotifier) {

	timer := time.NewTimer(time.Second * 60)
	defer timer.Stop()
	var wg sync.WaitGroup

	for {
		log.Debug("Posting HTLC Statistics")
		wg.Add(1)
		go func() {
			<-timer.C
			err := telegramNotifier.Notify(fmt.Sprintf("HTLC-Stats (1 Day): Settled %d, LinkFail: %d, ForwardFail %d, TotalForwards%d", stat.CounterWeek[stat.WeekDay].Settled,
				stat.CounterWeek[stat.WeekDay].LinkFails, stat.CounterWeek[stat.WeekDay].ForwardFails, stat.CounterWeek[stat.WeekDay].ForwardEvents))

			if err != nil {
				log.Error("htlc stats error", err)
			}
			timer.Reset(time.Second * 60)
		}()
		wg.Wait()

	}
}

func (stat *HtlcStatistics) DispatchHtlcStats(ctx context.Context, telegramNotifier *TelegramNotifier) {

	// Htlc statistics
	go func() {
		log.Debug("PostStats")
		stat.postStats(ctx, telegramNotifier)

		// release wait group for channel acceptor
		ctx.Value(ctxKeyWaitGroup).(*sync.WaitGroup).Done()
	}()
}
