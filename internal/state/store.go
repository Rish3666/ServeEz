// Package state implements the ServeEz object store — the single source of
// truth for all cluster state. It is a typed, JSON-schema-validated store
// (SQLite embedded for small clusters) with optimistic concurrency via
// resourceVersion tokens and CRD-equivalent schema registration.
//
// Design doc: AI Control/02 (Object Store Backing) + Orchestration/02 (State Store).
package state

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
)

// ErrNotFound is returned when an object does not exist.
var ErrNotFound = errors.New("object not found")

// ErrConflict is returned on a stale write (optimistic concurrency).
var ErrConflict = errors.New("resource version conflict")

// ErrValidation is returned when an object fails schema validation.
var ErrValidation = errors.New("validation failed")

// ErrKindUnknown is returned for unregistered object kinds.
var ErrKindUnknown = errors.New("unknown object kind")

// Store is the persistence interface for cluster objects.
type Store interface {
	// Create persists a new object. Returns the assigned resourceVersion.
	Create(ctx context.Context, obj *api.Object) (api.ResourceVersion, error)
	// Get fetches an object by kind + name.
	Get(ctx context.Context, kind, namespace, name string) (*api.Object, error)
	// List returns all objects of a kind (or all kinds when kind == "").
	List(ctx context.Context, kind, namespace string) ([]*api.Object, error)
	// Update writes obj iff its ResourceVersion matches the stored one.
	Update(ctx context.Context, obj *api.Object) (api.ResourceVersion, error)
	// Delete removes an object iff version matches.
	Delete(ctx context.Context, kind, namespace, name string, version api.ResourceVersion) error
	// Watch streams change events for objects matching the filter.
	Watch(ctx context.Context) (<-chan WatchEvent, error)
	// Close releases resources.
	Close() error
}

// WatchEvent is a single state change delivered to subscribers.
type WatchEvent struct {
	Kind   string
	Object *api.Object
	Action string // "create", "update", "delete"
	At     time.Time
}

// Schema describes a registered object kind.
type Schema struct {
	// Kind is the object type name, e.g. "Node".
	Kind string
	// Version is the schema version for this kind.
	Version string
	// Validate validates a decoded spec; returns an error describing the failure.
	Validate func(spec any) error
	// NewSpec returns a new empty spec value (a pointer) used to decode
	// stored JSON back into a typed struct. Optional for read-only kinds.
	NewSpec func() any
	// NewStatus returns a new empty status value (a pointer) used to decode
	// stored JSON back into a typed struct. Optional.
	NewStatus func() any
}

// Registry tracks registered object kinds and validates objects against them.
type Registry struct {
	mu      sync.RWMutex
	schemas map[string]*Schema // key: "Kind"
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{schemas: map[string]*Schema{}}
}

// Register adds a schema for an object kind (CRD-equivalent). Duplicate kinds
// are an error unless the same version is re-registered.
func (r *Registry) Register(s Schema) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s.Kind == "" {
		return fmt.Errorf("%w: kind required", ErrValidation)
	}
	if existing, ok := r.schemas[s.Kind]; ok && existing.Version != s.Version {
		return fmt.Errorf("kind %q already registered with version %q", s.Kind, existing.Version)
	}
	r.schemas[s.Kind] = &s
	return nil
}

// Lookup returns the schema for a kind, or nil.
func (r *Registry) Lookup(kind string) *Schema {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.schemas[kind]
}

// Kinds returns all registered kind names.
func (r *Registry) Kinds() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.schemas))
	for k := range r.schemas {
		out = append(out, k)
	}
	return out
}

// Validate checks an object against its registered schema (if any).
func (r *Registry) Validate(obj *api.Object) error {
	sc := r.Lookup(obj.Kind)
	if sc == nil {
		// Unknown kinds are allowed only when the registry is empty or the
		// object carries no schema version and the kind is not registered.
		// Here we enforce registration to keep the type system strict.
		return fmt.Errorf("%w: kind %q", ErrKindUnknown, obj.Kind)
	}
	obj.SchemaVersion = sc.Version
	if sc.Validate != nil {
		if err := sc.Validate(obj.Spec); err != nil {
			return fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}
	return nil
}

// DecodeSpec converts a raw decoded spec (map[string]any) into the typed spec
// struct registered for kind, using JSON round-trip. Returns the raw value
// unchanged if the kind has no typed decoder.
func (r *Registry) DecodeSpec(kind string, raw any) (any, error) {
	sc := r.Lookup(kind)
	if sc == nil || sc.NewSpec == nil {
		return raw, nil
	}
	target := sc.NewSpec()
	if raw == nil {
		return target, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, target); err != nil {
		return nil, err
	}
	return target, nil
}

// DecodeStatus is like DecodeSpec for the Status field.
func (r *Registry) DecodeStatus(kind string, raw any) (any, error) {
	sc := r.Lookup(kind)
	if sc == nil || sc.NewStatus == nil {
		return raw, nil
	}
	target := sc.NewStatus()
	if raw == nil {
		return target, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, target); err != nil {
		return nil, err
	}
	return target, nil
}

// newResourceVersion returns a random opaque version token.
func newResourceVersion() api.ResourceVersion {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return api.ResourceVersion(hex.EncodeToString(b))
}

// memStore is an in-memory Store implementation, used as the fallback when
// no persistent backend is configured (e.g. tests, single-node dev).
type memStore struct {
	mu      sync.RWMutex
	reg     *Registry
	objects map[string]*api.Object // key: kind + "/" + namespace + "/" + name
	watches []chan WatchEvent
	seq     uint64
}

// NewMemStore returns an in-memory store.
func NewMemStore() Store {
	return &memStore{objects: map[string]*api.Object{}}
}

// NewMemStoreWithRegistry returns an in-memory store that decodes typed
// Spec/Status fields using reg.
func NewMemStoreWithRegistry(reg *Registry) Store {
	return &memStore{reg: reg, objects: map[string]*api.Object{}}
}

func memKey(kind, namespace, name string) string {
	if namespace == "" {
		namespace = "default"
	}
	return kind + "/" + namespace + "/" + name
}

func (m *memStore) Create(ctx context.Context, obj *api.Object) (api.ResourceVersion, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := memKey(obj.Kind, obj.Namespace, obj.Name)
	if _, exists := m.objects[key]; exists {
		return "", fmt.Errorf("object %s already exists", key)
	}
	now := time.Now().UTC()
	obj.CreatedAt = now
	obj.UpdatedAt = now
	obj.ResourceVersion = newResourceVersion()
	clone := cloneObject(obj)
	m.objects[key] = clone
	m.emitLocked(clone, "create")
	return clone.ResourceVersion, nil
}

func (m *memStore) Get(ctx context.Context, kind, namespace, name string) (*api.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	obj, ok := m.objects[memKey(kind, namespace, name)]
	if !ok {
		return nil, ErrNotFound
	}
	return m.decodeObject(obj), nil
}

func (m *memStore) List(ctx context.Context, kind, namespace string) ([]*api.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*api.Object
	for _, obj := range m.objects {
		if kind != "" && obj.Kind != kind {
			continue
		}
		if namespace != "" && obj.Namespace != namespace {
			continue
		}
		out = append(out, m.decodeObject(obj))
	}
	return out, nil
}

func (m *memStore) Update(ctx context.Context, obj *api.Object) (api.ResourceVersion, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := memKey(obj.Kind, obj.Namespace, obj.Name)
	existing, ok := m.objects[key]
	if !ok {
		return "", ErrNotFound
	}
	if existing.ResourceVersion != obj.ResourceVersion {
		return "", ErrConflict
	}
	now := time.Now().UTC()
	obj.CreatedAt = existing.CreatedAt
	obj.UpdatedAt = now
	obj.ResourceVersion = newResourceVersion()
	clone := cloneObject(obj)
	m.objects[key] = clone
	m.emitLocked(clone, "update")
	return clone.ResourceVersion, nil
}

func (m *memStore) Delete(ctx context.Context, kind, namespace, name string, version api.ResourceVersion) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := memKey(kind, namespace, name)
	existing, ok := m.objects[key]
	if !ok {
		return ErrNotFound
	}
	if version != "" && existing.ResourceVersion != version {
		return ErrConflict
	}
	delete(m.objects, key)
	m.emitLocked(existing, "delete")
	return nil
}

func (m *memStore) Watch(ctx context.Context) (<-chan WatchEvent, error) {
	ch := make(chan WatchEvent, 256)
	m.mu.Lock()
	m.watches = append(m.watches, ch)
	m.mu.Unlock()
	go func() {
		<-ctx.Done()
		m.mu.Lock()
		defer m.mu.Unlock()
		for i, w := range m.watches {
			if w == ch {
				m.watches = append(m.watches[:i], m.watches[i+1:]...)
				break
			}
		}
		close(ch)
	}()
	return ch, nil
}

func (m *memStore) emitLocked(obj *api.Object, action string) {
	for _, w := range m.watches {
		select {
		case w <- WatchEvent{Kind: obj.Kind, Object: m.decodeObject(obj), Action: action, At: time.Now().UTC()}:
		default: // drop if the subscriber is slow
		}
	}
}

func (m *memStore) Close() error { return nil }

// cloneObject performs a deep copy via JSON round-trip, then re-decodes the
// Spec/Status into the typed structs registered for the object's kind (when a
// registry is present). Kept simple for now; switch to a schema-aware clone
// when performance demands it.
func cloneObject(obj *api.Object) *api.Object {
	var c api.Object
	b, _ := json.Marshal(obj)
	_ = json.Unmarshal(b, &c)
	return &c
}

// decodeObject re-encodes the given object's Spec/Status into typed structs.
func (m *memStore) decodeObject(obj *api.Object) *api.Object {
	c := cloneObject(obj)
	if m.reg == nil {
		return c
	}
	if spec, err := m.reg.DecodeSpec(c.Kind, c.Spec); err == nil {
		c.Spec = spec
	}
	if status, err := m.reg.DecodeStatus(c.Kind, c.Status); err == nil {
		c.Status = status
	}
	return c
}
