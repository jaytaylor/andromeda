package crawler

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"jaytaylor.com/andromeda/domain"
)

type Remote struct {
	Addr    string
	crawler *Crawler
}

func NewRemote(addr string, cfg *Config) *Remote {
	r := &Remote{
		Addr:    addr,
		crawler: New(cfg),
	}
	return r
}

func (r *Remote) Run(stopCh chan struct{}) {
	var result *domain.CrawlResult

	for {
		err := func() error {
			conn, err := grpc.Dial(r.Addr, grpc.WithInsecure())
			if err != nil {
				return fmt.Errorf("Dialing %v: %s", r.Addr, err)
			}
			defer conn.Close()

			rcsc := domain.NewRemoteCrawlerServiceClient(conn)
			ac, err := rcsc.Attach(context.Background())
			if err != nil {
				return fmt.Errorf("Getting AttachClient: %s", r.Addr, err)
			}

			if result != nil {
				log.Debugf("Sending previously unsent result=%# v", *result)
				if err = ac.Send(result); err != nil {
					// TODO: Requeue locally.
					return err
				}
			}

			for {
				select {
				case <-stopCh:
					return ErrStopRequested
				default:
				}

				entry, err := ac.Recv()
				if err != nil {
					return fmt.Errorf("receiving ToCrawlEntry: %s", err)
				}
				pkg, err := r.crawler.Do(&domain.Package{Path: entry.PackagePath}, stopCh)
				if err != nil {
				}
				result = domain.NewCrawlResult(pkg, err)
				if err = ac.Send(result); err != nil {
					// TODO: Requeue locally.
					return err
				}
				result = nil
			}
		}()
		if err != nil {
			if err == ErrStopRequested {
				log.WithField("addr", r.Addr).Debug("Remote shutting down")
				return
			}
			log.Errorf("%s", err)
		}
		// TODO: Use backoff instead of sleep.
		log.Debug("Sleeping for 10s..")
		select {
		case <-stopCh:
			return
		case <-time.After(10 * time.Second):
		}
	}
}
