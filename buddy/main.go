package main

import (
	"context"
	"net/http"

	"github.com/exadrift/tools/buddy/internal/config"
	"github.com/exadrift/tools/buddy/internal/ui"
	"github.com/gojargo/jargo/frames"
	"github.com/gojargo/jargo/pipeline"
	"github.com/gojargo/jargo/processor/aggregators"
	"github.com/gojargo/jargo/provider/kokoro"
	"github.com/gojargo/jargo/provider/openai/chat"
	"github.com/gojargo/jargo/provider/whispercpp"
	"github.com/gojargo/jargo/runner"
	"github.com/gojargo/jargo/transport"
	"github.com/gojargo/jargo-web/prebuilt"
)

func main() {
	// We want to serve production UI at /, but the runner always redirects / to /client/
	// The runner's setupFrontendRoutes adds a redirect handler that takes precedence
	// over our handler for / because it's added after.

	// The solution: set runner.Frontend to nil to skip the redirect, then manually add
	// the /client/ route with the prebuilt handler, and add our own handler for /.

	// Create a custom mux for our routes
	customMux := http.NewServeMux()

	// Add our handler for / that serves production UI
	customMux.Handle("/", ui.Handler())

	// Add the /client/ route with the prebuilt handler (for JS files)
	customMux.Handle("/client/", http.StripPrefix("/client", prebuilt.Handler()))

	// Set runner.App to our custom mux
	runner.App = customMux

	// Set runner.Frontend to nil to prevent setupFrontendRoutes from adding the redirect
	runner.Frontend = nil

	// Let the runner add its routes
	runner.Main(bot)
}

func bot(ctx context.Context, args runner.Arguments) error {
	worker, err := createWorker(ctx, args)
	if err != nil {
		return err
	}

	return worker.Run(ctx)
}

func createWorker(ctx context.Context, args runner.Arguments) (*pipeline.Worker, error) {
	cfg := config.Get()

	t, err := runner.CreateTransport(ctx, args, runner.TransportParams{
		WebRTC: transport.DefaultParams,
	})
	if err != nil {
		return nil, err
	}

	sttProvider := whispercpp.NewSTT(whispercpp.Config{
		BaseURL:    cfg.WHISPER_ENDPOINT,
		SampleRate: 16000,
		Language:   "",
	})

	ttsProvider := kokoro.NewTTS(kokoro.Config{
		BaseURL: cfg.KOKORO_ENDPOINT,
		Voice:   cfg.KOKORO_DEFAULT_VOICE,
	})

	llmProvider := chat.NewLLM(chat.LLMConfig{
		BaseURL: cfg.OPENAI_ENDPOINT,
		APIKey:  cfg.OPENAI_API_KEY,
		Model:   cfg.OPENAI_MODEL,
	})

	aggregator := aggregators.New(
		frames.NewLLMContext(cfg.DEFAULT_PROMPT),
	)

	pipe := pipeline.New(
		t.Input(),
		sttProvider,
		aggregator.User(),
		llmProvider,
		ttsProvider,
		t.Output(),
		aggregator.Assistant(),
	)

	return pipeline.NewWorker(pipe, pipeline.WorkerConfig{
		IdleTimeout: args.Base().PipelineIdleTimeout,
	}), nil
}
