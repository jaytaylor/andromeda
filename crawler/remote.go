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
	log.WithField("grpc-server-addr", r.Addr).Info("Remote starting")
	var (
		res    *domain.CrawlResult
		crawls int
	)

	for {
		err := func() error {
			if r.crawler.Config.MaxItems > 0 && crawls >= r.crawler.Config.MaxItems {
				log.WithField("session-crawls", crawls).WithField("max-items", r.crawler.Config.MaxItems).Info("Limit reached, crawl session ending")
				return ErrStopRequested
			}

			conn, err := grpc.Dial(r.Addr, grpc.WithInsecure())
			if err != nil {
				return fmt.Errorf("Dialing %v: %s", r.Addr, err)
			}
			defer conn.Close()

			rcsc := domain.NewRemoteCrawlerServiceClient(conn)
			ac, err := rcsc.Attach(context.Background())
			if err != nil {
				return fmt.Errorf("attaching: %s", r.Addr, err)
			}

			if res != nil {
				log.Debugf("Sending previously unsent res=%# v", *res)
				if err = ac.Send(res); err != nil {
					// TODO: Requeue locally.
					return err
				}
				res = nil
				crawls++
			}

			for {
				if r.crawler.Config.MaxItems > 0 && crawls >= r.crawler.Config.MaxItems {
					log.WithField("session-crawls", crawls).WithField("max-items", r.crawler.Config.MaxItems).Info("Limit reached, crawl session ending")
					return ErrStopRequested
				}

				select {
				case <-stopCh:
					return ErrStopRequested
				default:
				}

				log.WithField("session-crawls", crawls).Debug("Ready to receive next entry")
				entry, err := ac.Recv()
				if err != nil {
					return fmt.Errorf("receiving entry: %s", err)
				}
				res, err := r.crawler.Do(&domain.Package{Path: entry.PackagePath}, stopCh)
				if res == nil && err != nil {
					res = domain.NewCrawlResult(nil, err)
				} else if res.Error() == nil && err != nil {
					res.ErrMsg = err.Error()
				}
				if err = ac.Send(res); err != nil {
					// TODO: Requeue locally.
					return err
				}
				res = nil
				crawls++
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
