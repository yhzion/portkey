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
