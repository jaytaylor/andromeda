package crawler

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
)

const MaxMsgSize = 100000000 // 100MB.

var (
	RemoteCrawlerReadDeadlineDuration = 60 * time.Second
	SleepDuration                     = 10 * time.Second
)

type Remote struct {
	Addr        string
	DialOptions []grpc.DialOption
	crawler     *Crawler
}

func NewRemote(addr string, crawlCfg *Config) *Remote {
	r := &Remote{
		Addr:        addr,
		DialOptions: []grpc.DialOption{},
		crawler:     New(crawlCfg),
	}
	return r
}

func (r *Remote) Run(stopCh chan struct{}) {
	log.WithField("grpc-server-addr", r.Addr).Info("Remote starting")
	var (
		res    *domain.CrawlResult
		crawls int
		b      = newBackoff()
	)

	for {
		err := func() error {
			if r.crawler.Config.MaxItems > 0 && crawls >= r.crawler.Config.MaxItems {
				log.WithField("session-crawls", crawls).WithField("max-items", r.crawler.Config.MaxItems).Info("Limit reached, crawl session ending")
				return ErrStopRequested
			}

			conn, err := r.conn()
			if err != nil {
				return err
			}
			defer conn.Close()

			rcsc := domain.NewRemoteCrawlerServiceClient(conn)
			ctx, cancelFn := context.WithDeadline(context.Background(), time.Now().Add(RemoteCrawlerReadDeadlineDuration))
			ac, err := rcsc.Attach(ctx)
			if err != nil {
				return fmt.Errorf("attaching %v: %s", r.Addr, err)
			}

			b.Reset()

			if res != nil && res.Package != nil {
				log.WithField("pkg", res.Package.Path).Debugf("Sending previously unsent result")
				opRes, err := rcsc.Receive(ctx, res)
				if err != nil {
					return err
				}
				// TODO: consider using boolean field for success and make the msg not be
				//       error specific.
				if opRes.ErrMsg != "" {
					return fmt.Errorf(opRes.ErrMsg)
				}
				log.Debug("Successfully uploaded previous result")
				// if err = ac.Send(res); err != nil {
				// 	// TODO: Requeue locally.
				// 	return err
				// }
				res = nil
				crawls++
			}

			for {
				if r.crawler.Config.MaxItems > 0 && crawls >= r.crawler.Config.MaxItems {
					log.WithField("session-crawls", crawls).WithField("max-items", r.crawler.Config.MaxItems).Info("Limit reached, crawl session ending")
					return ErrStopRequested
				}

				var (
					entry        *domain.ToCrawlEntry
					notStoppedCh = make(chan struct{}, 1)
					recvCh       = make(chan error)
				)

				go func() {
					select {
					case <-stopCh:
						cancelFn()

					case <-notStoppedCh:
					}
				}()

				go func() {
					log.WithField("session-crawls", crawls).Debug("Ready to receive next entry")
					var err error
					entry, err = ac.Recv()
					recvCh <- err
				}()

				err = <-recvCh
				notStoppedCh <- struct{}{}
				if err != nil {
					if err == context.Canceled {
						return ErrStopRequested
					} else if err == context.DeadlineExceeded {
						return err
					} else {
						return fmt.Errorf("receiving entry: %s", err)
					}
				}

				res, err = r.crawler.Do(&domain.Package{Path: entry.PackagePath}, stopCh)

				if res == nil && err != nil {
					res = domain.NewCrawlResult(nil, err)
				} else if res.Error() == nil && err != nil {
					res.ErrMsg = err.Error()
				}

				if err = ac.Send(res); err != nil {
					// TODO: Requeue locally.
					return fmt.Errorf("sending result: %s", err)
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
			if err == context.DeadlineExceeded {
				continue
			}
		}
		// TODO: Use backoff instead of sleep.
		duration := b.NextBackOff()
		log.Debugf("Backoff: sleeping for %s..", duration)
		select {
		case <-stopCh:
			return
		case <-time.After(duration):
		}
	}
}

func (r *Remote) Enqueue(entries []*domain.ToCrawlEntry, opts *db.QueueOptions) (int, error) {
	if opts == nil {
		opts = db.NewQueueOptions()
	}

	conn, err := r.conn()
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	rcsc := domain.NewRemoteCrawlerServiceClient(conn)

	req := &domain.EnqueueRequest{
		Entries:         entries,
		Priority:        int32(opts.Priority),
		OnlyIfNotExists: opts.OnlyIfNotExists,
	}

	resp, err := rcsc.Enqueue(context.Background(), req)
	if err != nil {
		return 0, err
	}
	return int(resp.N), nil
}

func (r *Remote) Do(pkgs []string, stopCh chan struct{}) error {
	conn, err := r.conn()
	if err != nil {
		return err
	}
	defer conn.Close()

	rcsc := domain.NewRemoteCrawlerServiceClient(conn)

	for _, pkg := range pkgs {
		res, err := r.crawler.Do(&domain.Package{Path: pkg}, stopCh)
		if res == nil && err != nil {
			res = domain.NewCrawlResult(nil, err)
		} else if res.Error() == nil && err != nil {
			res.ErrMsg = err.Error()
		}

		resp, err := rcsc.Receive(context.Background(), res)
		if err != nil {
			return err
		}
		if resp.ErrMsg != "" {
			return fmt.Errorf("op response message: %s", resp.ErrMsg)
		}
		log.WithField("pkg", pkg).Debugf("Successfully transmitted crawl result")
	}
	return nil
}

func (r *Remote) conn() (*grpc.ClientConn, error) {
	if len(r.DialOptions) == 0 {
		log.Debug("Activated gRPC dial option grpc.WithInsecure() due to empty options")
		r.DialOptions = append(r.DialOptions, grpc.WithInsecure())
	}

	r.DialOptions = append(r.DialOptions, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(MaxMsgSize),
		grpc.MaxCallSendMsgSize(MaxMsgSize),
	))

	conn, err := grpc.Dial(r.Addr, r.DialOptions...)
	if err != nil {
		return nil, fmt.Errorf("Dialing %v: %s", r.Addr, err)
	}
	return conn, nil
}

func newBackoff() backoff.BackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 100 * time.Millisecond
	b.MaxElapsedTime = time.Duration(0)
	b.MaxInterval = 10 * time.Second
	b.Multiplier = 1.5
	b.RandomizationFactor = 0.5
	return b
}
