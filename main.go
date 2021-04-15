package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	flag "github.com/spf13/pflag"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

type hey struct {
	duration    time.Duration
	connections *int
	rate        *int
}

func main() {
	var periods []time.Duration
	var conns []int
	var rates []int
	var random int
	var u string

	flag.StringVar(&u, "url", "", "the url to send requests to, must include the scheme")
	flag.DurationSliceVar(&periods, "periods", []time.Duration{}, "the hey periods")
	flag.IntSliceVar(&conns, "connections", []int{}, "connections per period")
	flag.IntSliceVar(&rates, "rate", []int{}, "rate per period")
	flag.IntVar(&random, "random", 0, "if set, creates random X periods, each from 1m to 10m with random connections and rates")
	flag.Parse()

	if len(u) == 0 {
		log.Error().Msg("no url provided, exiting...")
		return
	}

	if random > 0 {
		for i := 0; i < random; i++ {
			periods = append(periods, generateRandomPeriod())
			conns = append(conns, generateRandomConnection())
			rates = append(rates, generateRandomRates())
		}
	}

	if len(periods) == 0 {
		log.Info().Msg("no periods provided, going to use a default 1m period")
		periods = append(periods, time.Minute)
	}

	parsedUrl, err := url.ParseRequestURI(u)
	if err != nil {
		log.Err(err).Msg("could not parse url, exiting...")
		return
	}

	heys := []hey{}
	for i, per := range periods {
		heys = append(heys, hey{
			duration: per,
			connections: func() *int {
				if len(conns) >= i+1 {
					return &conns[i]
				}

				return nil
			}(),
			rate: func() *int {
				if len(rates) >= i+1 {
					return &rates[i]
				}

				return nil
			}(),
		})
	}

	ctx, canc := context.WithCancel(context.Background())
	exitChan := make(chan struct{})
	sig := make(chan os.Signal, 1)

	go func() {
		for _, hey := range heys {
			args := []string{"-z", hey.duration.String()}
			if hey.connections != nil {
				args = append(args, "-c", fmt.Sprintf("%d", *hey.connections))
			}
			if hey.rate != nil {
				args = append(args, "-q", fmt.Sprintf("%d", *hey.rate))
			}
			args = append(args, parsedUrl.String())

			log.Info().Msg(fmt.Sprintf("hey %s", strings.Join(args, " ")))
			cmd := exec.CommandContext(ctx, "hey", args...)
			err := cmd.Run()
			if err != nil {
				if strings.Contains(err.Error(), "signal: killed") || errors.Is(err, context.Canceled) {
					log.Info().Msg("operation canceled")
					close(exitChan)
					return
				}

				log.Err(err).Msg("could not execute command, skipping...")
			} else {
				log.Info().Msg("finished")
			}
		}

		log.Info().Msg("all commands finished, exiting...")
		canc()
		close(sig)
		close(exitChan)
	}()

	// Graceful shutdown
	signal.Notify(sig, os.Interrupt)
	<-sig
	fmt.Println()

	canc()
	<-exitChan
	log.Info().Msg("good bye!")
}

func generateRandomPeriod() time.Duration {
	rand.Seed(time.Now().UnixNano())
	min := 1
	max := 5
	return time.Duration((rand.Intn(max-min) + min)) * time.Minute
}

func generateRandomConnection() int {
	rand.Seed(time.Now().UnixNano())
	min := 1
	max := 50
	return rand.Intn(max-min) + min
}

func generateRandomRates() int {
	rand.Seed(time.Now().UnixNano())
	min := 3
	max := 100
	return rand.Intn(max-min) + min
}
