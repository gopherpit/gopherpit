package timed // import "resenje.org/boltdbpool/timed"

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"resenje.org/boltdbpool"
)

var (
	ErrUnknownDB     = errors.New("unknown database")
	ErrUnknownPeriod = errors.New("unknown period")
)

type period int

const (
	_             = iota
	Hourly period = iota
	Daily
	Monthly
	Yearly
)

type Pool struct {
	pool   *boltdbpool.Pool
	series []string
	dir    string
	period period
	mu     *sync.Mutex
}

func New(dir string, p period, options *boltdbpool.Options) (*Pool, error) {
	series := []string{}
	switch p {
	case Hourly:
		matches, err := filepath.Glob(filepath.Join(dir, "??????", "??????????.db"))
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			match = filepath.Base(match)
			if len(match) >= 10 {
				series = append(series, match[:10])
			}
		}
	case Daily:
		matches, err := filepath.Glob(filepath.Join(dir, "??????", "????????.db"))
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			match = filepath.Base(match)
			if len(match) >= 8 {
				series = append(series, match[:8])
			}
		}
	case Monthly:
		matches, err := filepath.Glob(filepath.Join(dir, "??????.db"))
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			match = filepath.Base(match)
			if len(match) >= 6 {
				series = append(series, match[:6])
			}
		}
	case Yearly:
		matches, err := filepath.Glob(filepath.Join(dir, "????.db"))
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			match = filepath.Base(match)
			if len(match) >= 4 {
				series = append(series, match[:4])
			}
		}
	default:
		return nil, ErrUnknownPeriod
	}
	return &Pool{
		pool:   boltdbpool.New(options),
		series: series,
		dir:    dir,
		period: p,
		mu:     &sync.Mutex{},
	}, nil
}

func (p Pool) seriesFromTime(t time.Time) string {
	if p.period == Hourly {
		return t.Format("2006010215")
	}
	if p.period == Daily {
		return t.Format("20060102")
	}
	if p.period == Monthly {
		return t.Format("200601")
	}
	if p.period == Yearly {
		return t.Format("2006")
	}
	return ""
}

func (p Pool) pathFromSeries(series string) (path string) {
	if p.period == Hourly && len(series) == 10 {
		return filepath.Join(p.dir, series[:6], series+".db")
	}
	if p.period == Daily && len(series) == 8 {
		return filepath.Join(p.dir, series[:6], series+".db")
	}
	if p.period == Monthly && len(series) == 6 {
		return filepath.Join(p.dir, series+".db")
	}
	if p.period == Yearly && len(series) == 4 {
		return filepath.Join(p.dir, series+".db")
	}
	return
}

func (p Pool) connFromPath(path string) (c *boltdbpool.Connection, err error) {
	if path == "" {
		err = ErrUnknownDB
		return
	}
	return p.pool.Get(path)
}

func (p *Pool) NewConnection(t time.Time) (conn *Connection, err error) {
	series := p.seriesFromTime(t)
	path := p.pathFromSeries(series)
	c, err := p.pool.Get(path)
	if err != nil {
		return nil, err
	}

	found := false
	p.mu.Lock()
	for i := len(p.series) - 1; i >= 0; i-- {
		if p.series[i] == series {
			found = true
			break
		}
	}
	if !found {
		p.series = append(p.series, series)
	}
	p.mu.Unlock()

	return &Connection{
		Connection: c,
		pool:       p,
		series:     series,
	}, nil
}

func (p *Pool) GetConnection(t time.Time) (conn *Connection, err error) {
	series := p.seriesFromTime(t)
	path := p.pathFromSeries(series)
	if _, err = os.Stat(path); os.IsNotExist(err) {
		p.mu.Lock()
		for i := len(p.series) - 1; i >= 0; i-- {
			if p.series[i] == series {
				p.series = append(p.series[:i], p.series[i+1:]...)
				break
			}
		}
		p.mu.Unlock()
		err = ErrUnknownDB
		return
	} else if err != nil {
		return
	}
	c, err := p.pool.Get(path)
	if err != nil {
		return nil, err
	}
	return &Connection{
		Connection: c,
		pool:       p,
		series:     series,
	}, nil
}

func (p *Pool) NextConnection(t time.Time) (conn *Connection, err error) {
	path := ""
	series := p.seriesFromTime(t)
	for i := 0; i < len(p.series); i++ {
		s := p.series[i]
		if s > series {
			path = p.pathFromSeries(s)
			if _, err = os.Stat(path); os.IsNotExist(err) {
				p.mu.Lock()
				p.series = append(p.series[:i], p.series[i+1:]...)
				p.mu.Unlock()
				continue
			} else if err != nil {
				return
			}
			series = s
			break
		}
	}
	if path == "" {
		err = ErrUnknownDB
		return
	}
	c, err := p.pool.Get(path)
	if err != nil {
		return nil, err
	}
	return &Connection{
		Connection: c,
		pool:       p,
		series:     series,
	}, nil
}

func (p *Pool) PrevConnection(t time.Time) (conn *Connection, err error) {
	path := ""
	series := p.seriesFromTime(t)
	for i := len(p.series) - 1; i >= 0; i-- {
		s := p.series[i]
		if s < series {
			path = p.pathFromSeries(s)
			if _, err = os.Stat(path); os.IsNotExist(err) {
				p.mu.Lock()
				p.series = append(p.series[:i], p.series[i+1:]...)
				p.mu.Unlock()
				continue
			} else if err != nil {
				return
			}
			series = s
			break
		}
	}
	if path == "" {
		err = ErrUnknownDB
		return
	}
	c, err := p.pool.Get(path)
	if err != nil {
		return nil, err
	}
	return &Connection{
		Connection: c,
		pool:       p,
		series:     series,
	}, nil
}

type Connection struct {
	*boltdbpool.Connection
	pool   *Pool
	series string
}

func (c Connection) Next() (*Connection, error) {
	c.pool.mu.Lock()
	defer c.pool.mu.Unlock()

	for i := 0; i < len(c.pool.series)-1; i++ {
		s := c.pool.series[i]
		if s == c.series {
			ns := c.pool.series[i+1]
			nc, err := c.pool.connFromPath(c.pool.pathFromSeries(ns))
			if err != nil {
				return nil, err
			}
			return &Connection{
				Connection: nc,
				pool:       c.pool,
				series:     ns,
			}, nil
		}
	}
	return nil, ErrUnknownDB
}

func (c Connection) Prev() (*Connection, error) {
	c.pool.mu.Lock()
	defer c.pool.mu.Unlock()

	for i := len(c.pool.series) - 1; i > 0; i-- {
		s := c.pool.series[i]
		if s == c.series {
			ns := c.pool.series[i-1]
			nc, err := c.pool.connFromPath(c.pool.pathFromSeries(ns))
			if err != nil {
				return nil, err
			}
			return &Connection{
				Connection: nc,
				pool:       c.pool,
				series:     ns,
			}, nil
		}
	}
	return nil, ErrUnknownDB
}
