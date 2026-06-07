package config

func (c *Config) AddHost(h Host) {
	c.Hosts = append(c.Hosts, h)
}

func (c *Config) UpdateHost(index int, h Host) {
	if index >= 0 && index < len(c.Hosts) {
		c.Hosts[index] = h
	}
}

func (c *Config) RemoveHost(index int) {
	if index >= 0 && index < len(c.Hosts) {
		c.Hosts = append(c.Hosts[:index], c.Hosts[index+1:]...)
	}
}
