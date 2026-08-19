package main

import "time"

type clipboardLease struct {
	read    func() string
	write   func(string)
	now     func() time.Time
	value   string
	expires time.Time
}

func (c *clipboardLease) copy(value string, lifetime time.Duration) {
	c.write(value)
	c.value = value
	if lifetime <= 0 {
		c.expires = time.Time{}
		return
	}
	c.expires = c.now().Add(lifetime)
}

func (c *clipboardLease) tick() bool {
	if c.value == "" || c.expires.IsZero() || c.now().Before(c.expires) {
		return false
	}
	if c.read() == c.value {
		c.write("")
	}
	c.value = ""
	c.expires = time.Time{}
	return true
}

func (c *clipboardLease) clear() {
	if c.value != "" && c.read() == c.value {
		c.write("")
	}
	c.value = ""
	c.expires = time.Time{}
}
