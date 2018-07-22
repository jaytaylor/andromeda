package feed

import (
	"fmt"
	"sync"

	"gigawatt.io/errorlib"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
)

// DefaultSchedule sets the default feed refresh cron schedule.
var DefaultSchedule = "0 */5 * * * *"

type Config struct {
	Sources  []DataSource
	Schedule string
}

type Feed struct {
	Config Config
	ch     chan string
	cron   *cron.Cron
	mu     sync.Mutex
}

func New(cfg Config) *Feed {
	if cfg.Schedule == "" {
		cfg.Schedule = DefaultSchedule
	}
	f := &Feed{
		Config: cfg,
	}
	return f
}

func (f *Feed) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.ch != nil {
		return errorlib.NotRunningError
	}

	f.ch = make(chan string, 100)
	f.cron = cron.New()
	for _, s := range f.Config.Sources {
		func(s DataSource) {
			log.WithField("data-source", fmt.Sprintf("%T", s)).WithField("schedule", f.Config.Schedule).Debug("Adding feed to cron")
			f.cron.AddFunc(f.Config.Schedule, func() {
				log.WithField("data-source", fmt.Sprintf("%T", s)).Debug("Starting refresh")
				possiblePkgs, err := s.Refresh()
				if err != nil {
					log.WithField("data-source", fmt.Sprintf("%T", s)).Errorf("Refresh failed: %s", err)
					return
				}
				if len(possiblePkgs) == 0 {
					return
				}
				for _, p := range possiblePkgs {
					f.ch <- p
				}
			})
		}(s)
	}
	f.cron.Start()

	return nil
}

func (f *Feed) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.ch == nil {
		return errorlib.AlreadyRunningError
	}
	f.cron.Stop()
	f.cron = nil
	close(f.ch)
	f.ch = nil

	return nil
}

func (f *Feed) Channel() <-chan string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ch
}
