package main

import (
	"context"
	"fmt"

	"github.com/exadrift/tools/buddy/internal/config"
	"github.com/gojargo/jargo/frames"
	"github.com/gojargo/jargo/pipeline"
	"github.com/gojargo/jargo/processor/aggregators"
	"github.com/gojargo/jargo/provider/kokoro"
	"github.com/gojargo/jargo/provider/openai/chat"
	"github.com/gojargo/jargo/provider/whispercpp"
	"github.com/gojargo/jargo/runner"
	"github.com/gojargo/jargo/transport"
)

func main() {
	fmt.Println("hello")
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
