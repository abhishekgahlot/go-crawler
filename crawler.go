package crawler

import (
	"net/http"
	"net/url"
	"time"
)

type Crawler struct {
	u       *url.URL
	limit   int
	worker  int
	ch_url  chan string
	ch_err  chan error
	ch_done chan bool
	fetcher Fetcher
	filters []Filter
	scopes  []Filter
}

type Config struct {
	Url     string
	Limit   int
	Client  *http.Client
	Filters []Filter
	Scopes  []Filter
}

func New(config *Config) (*Crawler, error) {
	u, err := url.Parse(config.Url)

	if err != nil {
		return nil, err
	}

	limit := config.Limit
	client := config.Client
	filters := config.Filters
	scopes := config.Scopes

	//default limit - 10
	if limit == 0 {
		limit = 10
	}

	//default client - timeout 10s
	if client == nil {
		client = &http.Client{
			Timeout: time.Second * 10,
		}
	}

	anchor_picker := &PickerAttr{
		TagName: "a",
		Attr:    "href",
	}

	fetcher := Fetcher{
		Client: client,
		Picker: anchor_picker,
	}

	//default filters - url + unique
	if filters == nil {
		//http url filter
		filter_url := &FilterUrl{}

		//unique filter
		filter_unique := &FilterUnique{
			[]*url.URL{u},
		}

		filters = []Filter{filter_url, filter_unique}
	}

	//default scope - url + unique + same host
	if scopes == nil {
		//http url filter
		scope_url := &FilterUrl{}

		//unique filter
		scope_unique := &FilterUnique{
			[]*url.URL{u},
		}

		//same host filter
		scope_samehost := &FilterSameHost{
			u,
		}

		scopes = []Filter{scope_url, scope_unique, scope_samehost}
	}

	return &Crawler{
		u:       u,
		limit:   limit,
		ch_url:  make(chan string),
		ch_err:  make(chan error),
		ch_done: make(chan bool),
		fetcher: fetcher,
		filters: filters,
		scopes:  scopes,
	}, nil
}

type receiver_url func(string)
type receiver_err func(error)

func (c *Crawler) Start(r_url receiver_url, r_err receiver_err) {
	go c.crawl(c.u)

loop:
	for {
		select {
		case url := <-c.ch_url:
			r_url(url)
		case err := <-c.ch_err:
			r_err(err)
		case <-c.ch_done:
			break loop
		}
	}

	return
}

func (c *Crawler) crawl(u *url.URL) {
	c.worker++

	urls, err := c.fetcher.Fetch(u)

	if err != nil {
		c.ch_err <- err
	} else {
		for _, u_child := range urls {
			if c.isDone() {
				return
			}

			//check filters
			if c.checkFilters(c.filters, u_child) {
				c.ch_url <- u_child.String()
				c.limit--

				if c.limit <= 0 {
					// println("done by limit")

					if !c.isDone() {
						close(c.ch_done)
					}
					return
				}
			}

			//check scope for next crawl
			//yeah crawl but dont worry it's faster than run !
			if c.checkFilters(c.scopes, u_child) {
				go c.crawl(u_child)
			}
		}
	}

	//need to add delay here ?
	c.worker--
	if c.worker == 0 {
		// println("done by all worker done")

		if !c.isDone() {
			close(c.ch_done)
		}
	}
}

func (c *Crawler) checkFilters(filters []Filter, u *url.URL) bool {
	for _, f := range filters {
		if !f.Filter(u) {
			return false
		}
	}

	return true
}

func (c *Crawler) isDone() bool {
	select {
	case <-c.ch_done:
		return true
	default:
		return false
	}
}
