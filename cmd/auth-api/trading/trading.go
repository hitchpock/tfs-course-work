package trading

import (
	"context"
	"io"
	"sync"
	"time"

	"gitlab.com/hitchpock/tfs-course-work/cmd/auth-api/handlers"
	"gitlab.com/hitchpock/tfs-course-work/internal/fintech"
	"gitlab.com/hitchpock/tfs-course-work/internal/robot"
	"gitlab.com/hitchpock/tfs-course-work/pkg/log"
	"google.golang.org/grpc"
)

const (
	timeToSleep = 3
)

type Process struct {
	conn         *grpc.ClientConn
	logger       log.Logger
	robotStorage robot.Storage
	wsocket      *handlers.WSClients
}

func NewProcess(conn *grpc.ClientConn, logger log.Logger, storage robot.Storage, ws *handlers.WSClients) *Process {
	process := &Process{
		conn:         conn,
		logger:       logger,
		robotStorage: storage,
		wsocket:      ws,
	}

	return process
}

func (p *Process) StartTrading() {
	client := fintech.NewTradingServiceClient(p.conn)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), timeToSleep*time.Second)

		robots, err := p.robotStorage.FindToTrading()
		if err != nil {
			p.logger.Warnw("func robotStorage.FindToTrading return with error", "error", err)
		}

		tickerRobots := make(map[string][]robot.Robot)
		for _, rob := range robots {
			tickerRobots[rob.Ticker] = append(tickerRobots[rob.Ticker], rob)
		}

		tickers := make([]string, 0)
		for key := range tickerRobots {
			tickers = append(tickers, key)
		}

		tickerChan := make(map[string](chan *fintech.PriceResponse))
		for _, ticker := range tickers {
			tickerChan[ticker] = p.readService(ctx, client, ticker)
		}

		tickerChans := make(map[string]([]chan *fintech.PriceResponse))
		for ticker, value := range tickerChan {
			tickerChans[ticker] = separateChan(value, len(tickerRobots[ticker]))
		}

		var wg sync.WaitGroup

		wg.Add(len(robots))

		for key, value := range tickerChans {
			for index, ch := range value {
				go p.Trade(tickerRobots[key][index], ch, &wg)
			}
		}

		wg.Wait()
		cancel()
	}
}

func (p *Process) readService(ctx context.Context, client fintech.TradingServiceClient, ticker string) chan *fintech.PriceResponse {
	out := make(chan *fintech.PriceResponse)

	req := fintech.PriceRequest{Ticker: ticker}

	resp, err := client.Price(context.Background(), &req)
	if err != nil {
		p.logger.Warnw("grpc func Price return with error", "error", err)
	}

	go func() {
		defer close(out)

		for {
			price, err := resp.Recv()
			if err != nil {
				if err == io.EOF {
					p.logger.Warn("grpc channel is closed")
					break
				}

				p.logger.Warnw("grpc resp.Recv() return with error", "error", err)

				break
			}

			select {
			case <-ctx.Done():
				return

			default:
				out <- price
			}
		}
	}()

	return out
}

func separateChan(in chan *fintech.PriceResponse, number int) []chan *fintech.PriceResponse {
	outs := make([]chan *fintech.PriceResponse, 0)

	for i := 0; i < number; i++ {
		ch := make(chan *fintech.PriceResponse)
		outs = append(outs, ch)
	}

	go func() {
		defer closeChan(outs)

		for price := range in {
			for _, ch := range outs {
				ch := ch
				ch <- price
			}
		}
	}()

	return outs
}

func (p *Process) Trade(rob robot.Robot, in chan *fintech.PriceResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	for price := range in {
		if rob.IsBuying && price.BuyPrice < rob.BuyPrice {
			rob.Buy(price.BuyPrice)

			if err := p.robotStorage.Trade(&rob); err != nil {
				p.logger.Warnw("func robotTorage.Trade return with error", "error", err)
			}

			p.wsocket.Broadcast(rob.RobotID)
		} else if !rob.IsBuying && price.SellPrice > rob.SellPrice {
			rob.Sell(price.SellPrice)

			if err := p.robotStorage.Trade(&rob); err != nil {
				p.logger.Warnw("func robotTorage.Trade return with error", "error", err)
			}

			p.wsocket.Broadcast(rob.RobotID)
		}
	}
}

func closeChan(chans []chan *fintech.PriceResponse) {
	for _, ch := range chans {
		close(ch)
	}
}
