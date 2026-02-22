package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/stopwatch"
	"github.com/karavanix/karavantrack-api-server/pkg/utils"
	"github.com/uptrace/bun"
)

type KeyType interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~string
}

type ModelType[K KeyType] interface {
	Key() K
	Code() string
}

type DictionaryOptions[K KeyType, M ModelType[K]] func(*BaseDictionary[K, M])

type BaseDictionary[K KeyType, M ModelType[K]] struct {
	db          bun.IDB
	Items       []M
	ItemsBySlug map[string]M
	ItemsByKey  map[K]M
	mu          sync.RWMutex
	lastLoad    time.Time
	itemsTTL    time.Duration
	reloadable  bool
	// add additional enhancments here
	onLoad        func([]M) error
	relationships []string
	orderBy       string
	orderDir      string
}

func NewDictionary[K KeyType, M ModelType[K]](db bun.IDB, opts ...DictionaryOptions[K, M]) *BaseDictionary[K, M] {
	b := &BaseDictionary[K, M]{
		db:          db,
		ItemsByKey:  make(map[K]M),
		ItemsBySlug: make(map[string]M),
		itemsTTL:    time.Hour * 24,
		reloadable:  true,
	}

	for _, opt := range opts {
		opt(b)
	}

	if err := b.Load(context.Background()); err != nil {
		slog.ErrorContext(context.Background(), "failed to load BaseDictionary", "error.message", err.Error())
	}

	return b
}

func WithReloadable[K KeyType, M ModelType[K]](reloadable bool) func(*BaseDictionary[K, M]) {
	return func(o *BaseDictionary[K, M]) { o.reloadable = reloadable }
}

func WithItemsTTL[K KeyType, M ModelType[K]](itemsTTL time.Duration) func(*BaseDictionary[K, M]) {
	return func(o *BaseDictionary[K, M]) { o.itemsTTL = itemsTTL }
}

func WithOnLoad[K KeyType, M ModelType[K]](onLoad func([]M) error) DictionaryOptions[K, M] {
	return func(o *BaseDictionary[K, M]) {
		o.onLoad = onLoad
	}
}

func WithRelationships[K KeyType, M ModelType[K]](relationships ...string) DictionaryOptions[K, M] {
	return func(o *BaseDictionary[K, M]) {
		o.relationships = relationships
	}
}

func WithOrderBy[K KeyType, M ModelType[K]](orderBy string, orderDir string) DictionaryOptions[K, M] {
	return func(o *BaseDictionary[K, M]) {
		o.orderBy = orderBy
		o.orderDir = orderDir
	}
}

func (d *BaseDictionary[K, M]) Load(ctx context.Context) error {
	var models []M
	var model M

	watch := stopwatch.Start(fmt.Sprintf("Loading BaseDictionary %s", utils.GetTemplateName[M]()))
	query := d.db.NewSelect().Model(&models)

	if len(d.relationships) > 0 {
		for _, relation := range d.relationships {
			query = query.Relation(relation)
		}
	}

	if d.orderBy != "" {
		query = query.Order(fmt.Sprintf("%s %s NULLS LAST", d.orderBy, d.orderDir))
	}

	if err := query.Scan(ctx); err != nil {
		return Error(err, model)
	}

	if d.onLoad != nil {
		if err := d.onLoad(models); err != nil {
			return err
		}
	}

	d.mu.Lock()
	d.Items = models

	for _, m := range models {
		d.ItemsByKey[m.Key()] = m

		if m.Code() == "" {
			continue
		}

		d.ItemsBySlug[m.Code()] = m
	}
	d.mu.Unlock()

	d.lastLoad = time.Now()

	time.Sleep(time.Second)

	watch.Stop()

	return nil
}

func (d *BaseDictionary[K, M]) IsExpired() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return time.Since(d.lastLoad) > d.itemsTTL
}

func (d *BaseDictionary[K, M]) ReloadIfNeeded(ctx context.Context) {
	if d.IsExpired() && d.reloadable {
		if err := d.Load(ctx); err != nil {
			slog.WarnContext(ctx, "failed to load BaseDictionary", slog.String("error.message", err.Error()))
		}
	}
}

func (d *BaseDictionary[K, M]) Keys(ctx context.Context) ([]K, error) {
	d.ReloadIfNeeded(ctx)

	d.mu.RLock()
	defer d.mu.RUnlock()

	var keys []K

	for k := range d.ItemsByKey {
		keys = append(keys, k)
	}

	return keys, nil
}

func (d *BaseDictionary[K, M]) Values(ctx context.Context) ([]M, error) {
	d.ReloadIfNeeded(ctx)

	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.Items, nil
}

func (d *BaseDictionary[K, M]) GetByKey(ctx context.Context, key K) (M, error) {
	d.ReloadIfNeeded(ctx)

	d.mu.RLock()
	defer d.mu.RUnlock()

	var item M
	var ok bool

	if item, ok = d.ItemsByKey[key]; !ok {
		return item, inerr.NewErrNotFound(utils.GetTemplateName[M]())
	}

	return item, nil
}

func (d *BaseDictionary[K, M]) GetByCode(ctx context.Context, slug string) (M, error) {
	d.ReloadIfNeeded(ctx)

	d.mu.RLock()
	defer d.mu.RUnlock()

	var item M
	var ok bool

	if item, ok = d.ItemsBySlug[slug]; !ok {
		return item, inerr.NewErrNotFound(utils.GetTemplateName[M]())
	}

	return item, nil
}

func (d *BaseDictionary[K, M]) Len(ctx context.Context) int {
	d.ReloadIfNeeded(ctx)

	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.Items)
}
