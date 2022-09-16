package wrapper

import "io"

type Closer struct {
	closers []io.Closer
	err     error
}

func (c *Closer) Add(cl io.Closer) {
	c.closers = append(c.closers, cl)
}

func (c *Closer) Close() error {
	for _, f := range c.closers {
		err := f.Close()
		if err != nil && c.err == nil {
			c.err = err
		}
	}

	return c.err
}
