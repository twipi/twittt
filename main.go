package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"github.com/spf13/pflag"
	twicmdhttp "github.com/twipi/twipi/twicmd/http"
	"github.com/twipi/twittt/service"
	"golang.org/x/sync/errgroup"
	"libdb.so/hserve"
)

var (
	listenAddr = ":8080"
)

func init() {
	pflag.StringVarP(&listenAddr, "listen-addr", "l", listenAddr, "address to listen on")
	pflag.Parse()
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	logger := slog.Default()

	os.Exit(start(ctx, logger))
}

func start(ctx context.Context, logger *slog.Logger) int {
	errg, ctx := errgroup.WithContext(ctx)

	svc := service.NewService(logger)
	errg.Go(func() error { return svc.Start(ctx) })

	handler := twicmdhttp.NewHandler(svc, logger.With("component", "http"))
	errg.Go(func() error {
		<-ctx.Done()
		if err := handler.Close(); err != nil {
			logger.Error(
				"failed to close http service handler",
				"err", err)
		}
		return ctx.Err()
	})

	errg.Go(func() error {
		r := http.NewServeMux()
		r.Handle("GET /health", http.HandlerFunc(healthCheck))
		r.Handle("/", handler)

		logger.Info(
			"listening via HTTP",
			"addr", listenAddr)

		if err := hserve.ListenAndServe(ctx, listenAddr, r); err != nil {
			logger.Error(
				"failed to listen and serve",
				"err", err)
			return err
		}

		return ctx.Err()
	})

	if err := errg.Wait(); err != nil {
		logger.Error(
			"service error",
			"err", err)
		return 1
	}

	return 0
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
