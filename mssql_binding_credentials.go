package main

type MssqlBindingCredentials struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
}
