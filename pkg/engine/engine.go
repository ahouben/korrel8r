// package engine implements generic correlation logic to correlate across domains.
package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"golang.org/x/exp/maps"
)

// Engine combines a set of domains and a set of rules, so it can perform correlation.
type Engine struct {
	stores        map[string]korrel8r.Store
	domains       map[string]korrel8r.Domain
	rules         []korrel8r.Rule
	templateFuncs map[string]any
}

func New() *Engine {
	return &Engine{
		stores:        map[string]korrel8r.Store{},
		domains:       map[string]korrel8r.Domain{},
		templateFuncs: map[string]any{},
	}
}

// Domain returns the named domain or nil if not found.
func (e *Engine) Domain(name string) korrel8r.Domain { return e.domains[name] }
func (e *Engine) DomainErr(name string) (korrel8r.Domain, error) {
	if d := e.Domain(name); d != nil {
		return d, nil
	}
	return nil, fmt.Errorf("domain not found: %v", name)
}

// Domains returns a list of known domains.
func (e *Engine) Domains() (domains []korrel8r.Domain) { return maps.Values(e.domains) }

// Store returns the default store for domain, or nil if not found.
func (e *Engine) Store(name string) korrel8r.Store { return e.stores[name] }
func (e *Engine) StoreErr(name string) (korrel8r.Store, error) {
	if s := e.Store(name); s != nil {
		return s, nil
	}
	return nil, fmt.Errorf("store not found: %v", name)
}

// TemplateFuncser can be implemented by Domain or Store implementations to contribute
// domain-specific template functions to template rules generated by the Engine.
// See text/template.Template.Funcs for details.
type TemplateFuncser interface{ TemplateFuncs() map[string]any }

// AddDomain domain and corresponding store, store may be nil.
func (e *Engine) AddDomain(d korrel8r.Domain, s korrel8r.Store) {
	e.domains[d.String()] = d
	if s != nil {
		e.stores[d.String()] = s
	}
	// Stores and Domains implement TemplateFuncser if they provide template helper functions
	// for use by rules.
	for _, v := range []any{d, s} {
		if tf, ok := v.(TemplateFuncser); ok {
			maps.Copy(e.templateFuncs, tf.TemplateFuncs())
		}
	}
}

// Class parses a full 'domain/class' name and returns the class.
func (e *Engine) Class(name string) (korrel8r.Class, error) {
	d, c, ok := strings.Cut(name, "/")
	if !ok || c == "" || d == "" {
		return nil, fmt.Errorf("invalid class name: %v", name)
	}
	domain, err := e.DomainErr(d)
	if err != nil {
		return nil, err
	}
	class := domain.Class(c)
	if class == nil {
		return nil, fmt.Errorf("unknown class in domain %v: %v", d, c)
	}
	return class, nil
}

func (e *Engine) Rules() []korrel8r.Rule { return e.rules }

func (e *Engine) AddRules(rules ...korrel8r.Rule) { e.rules = append(e.rules, rules...) }

// Graph creates a new graph of the rules and classes of this engine.
func (e *Engine) Graph() *graph.Graph { return graph.NewData(e.rules...).NewGraph() }

// TemplateFuncs returns template helper functions for stores and domains known to this engine.
// See text/template.Template.Funcs
func (e *Engine) TemplateFuncs() map[string]any { return e.templateFuncs }

// Get finds the store for the query.Class() and gets into result.
func (e *Engine) Get(ctx context.Context, class korrel8r.Class, query korrel8r.Query, result korrel8r.Appender) error {
	store, err := e.StoreErr(class.Domain().String())
	if err != nil {
		return err
	}
	return store.Get(ctx, query, result)
}

func (e *Engine) Follower(ctx context.Context) *Follower { return &Follower{Engine: e, Context: ctx} }
