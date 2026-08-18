// Comando server sobe a API de clima por CEP.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"desafio-labs-go-1/internal/api"
	"desafio-labs-go-1/internal/cep"
	"desafio-labs-go-1/internal/weather"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	chave := os.Getenv("WEATHER_API")
	if chave == "" {
		// Melhor a revisão não subir do que subir devolvendo 500 em todo request.
		log.Error("WEATHER_API nao configurada")
		os.Exit(1)
	}

	porta := os.Getenv("PORT") // injetada pelo Cloud Run
	if porta == "" {
		porta = "8080"
	}

	handler := api.NewHandler(cep.NewClient(), weather.NewClient(chave))

	srv := &http.Server{
		Addr:              ":" + porta,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Cloud Run manda SIGTERM antes de recolher a instância; sair sem drenar
	// derruba requisição em voo.
	parar := make(chan os.Signal, 1)
	signal.Notify(parar, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info("servidor no ar", "porta", porta)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("servidor caiu", "erro", err)
			os.Exit(1)
		}
	}()

	<-parar
	log.Info("encerrando")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("encerramento forcado", "erro", err)
	}
}
