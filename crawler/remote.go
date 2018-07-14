package crawler

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
)

const MaxMsgSize = 50000000 // 50MB.

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
			ac, err := rcsc.Attach(context.Background())
			if err != nil {
				return fmt.Errorf("attaching %v: %s", r.Addr, err)
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

func (r *Remote) Enqueue(entries []*domain.ToCrawlEntry, priority ...int) (int, error) {
	conn, err := r.conn()
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	rcsc := domain.NewRemoteCrawlerServiceClient(conn)

	if len(priority) == 0 {
		priority = []int{db.DefaultQueuePriority}
	}
	req := &domain.EnqueueRequest{
		Entries:  entries,
		Priority: int32(priority[0]),
	}

	resp, err := rcsc.Enqueue(context.Background(), req)
	if err != nil {
		return 0, err
	}
	return int(resp.N), nil
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
