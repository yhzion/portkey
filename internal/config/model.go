package config

const DefaultPort = 22

type Host struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	LastUsed string `json:"lastUsed,omitempty"`
}

type Config struct {
	Hosts []Host `json:"hosts"`
}

// Clone returns a deep copy of c. The returned Config's Hosts slice is backed
// by a new array, so mutations to the clone (append, element assignment) do
// not affect the original and vice versa.
func (c *Config) Clone() *Config {
	if c == nil {
		return &Config{Hosts: []Host{}}
	}
	hosts := make([]Host, len(c.Hosts))
	copy(hosts, c.Hosts)
	return &Config{Hosts: hosts}
}
