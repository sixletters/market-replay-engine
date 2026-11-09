package marketdata

import (
	"context"
	"fmt"

	"sixletters/back-to-the-past/internal/model"
)

// Source is a long-running feed of normalized market events.
type Source interface {
	Markets() []model.MarketKey
	Run(ctx context.Context, out chan<- Event) error
}

// SourceConfig configures a source factory.
type SourceConfig struct {
	Markets []model.MarketKey
	Options map[string]string
}

// SourceFactory creates a source from a generic config.
type SourceFactory func(SourceConfig) (Source, error)

// Registry resolves source factories by venue/source name.
type Registry struct {
	factories map[string]SourceFactory
}

func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]SourceFactory)}
}

func (r *Registry) Register(name string, factory SourceFactory) {
	r.factories[name] = factory
}

func (r *Registry) New(name string, cfg SourceConfig) (Source, error) {
	factory, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("marketdata: unknown source %q", name)
	}
	return factory(cfg)
}
