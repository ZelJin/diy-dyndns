package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jasonlvhit/gocron"
	"github.com/spf13/viper"
)

type Config struct {
	Domains []DomainConfig `mapstructure:"domains"`
}

type DomainConfig struct {
	Domain     string   `mapstructure:"domain"`
	Subdomains []string `mapstructure:"subdomains"`
}

// DomainRecord contains information about the domain.
type DomainRecord struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	Priority int    `json:"priority"`
	Port     int    `json:"port"`
	Weight   int    `jsin:"weight"`
}

// DomainRecordResponse is a response of the Digital Ocean API
type DomainRecordResponse struct {
	DomainRecords []DomainRecord `json:"domain_records"`
	Links         interface{}    `json:"links"`
	Meta          interface{}    `jsin:"meta"`
}

var (
	config *Config
)

func init() {
	var err error
	config, err = ParseConfig()
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
}

func main() {
	token := os.Getenv("DO_TOKEN")
	if token != "" {
		for _, domainConfig := range config.Domains {
			gocron.Every(10).Minutes().Do(func() { CheckDomain(&domainConfig, token) })
			<-gocron.Start()
		}
	} else {
		os.Stderr.WriteString("Env variables are not configured correctly.\n")
		os.Exit(1)
	}
}

// CheckDomain checks the IP address assigned to the domain and its subdomains,
// compares it to the real IP address of the server and
// orders to change the DNS record if needed.
func CheckDomain(config *DomainConfig, token string) {
	externalIP, err := GetExternalIP()
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	os.Stdout.WriteString("External IP: " + externalIP + "\n")
	domainRecords, err := GetDomainRecords(config.Domain, token)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	for _, record := range domainRecords {
		CheckRecord(config, record, "@", externalIP, token)
		for _, subdomain := range config.Subdomains {
			CheckRecord(config, record, subdomain, externalIP, token)
		}
	}
}

func CheckRecord(config *DomainConfig, record DomainRecord, recordName string, externalIP string, token string) {
	if record.Type == "A" && record.Name == recordName {
		os.Stdout.WriteString(record.Name + " " + record.Data + "\n")
		if externalIP != record.Data {
			SetDomainRecord(config.Domain, record.ID, externalIP, token)
		}
	}
}

// GetExternalIP checks the external IP of the server
// using the external service
func GetExternalIP() (string, error) {
	res, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// GetDomainRecords queries Digital Ocean API for DNS records for a particular domain
func GetDomainRecords(domain string, token string) ([]DomainRecord, error) {
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET",
		"https://api.digitalocean.com/v2/domains/"+domain+"/records",
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var domainRecordsResponse DomainRecordResponse
	if err = json.Unmarshal(body, &domainRecordsResponse); err != nil {
		return nil, err
	}
	return domainRecordsResponse.DomainRecords, nil
}

// SetDomainRecord utilizes Digital Ocean API to set a DNS record
func SetDomainRecord(domain string, recordID int, IP string, token string) error {
	client := &http.Client{}
	jsonPayload, _ := json.Marshal(map[string]string{"data": IP})
	req, err := http.NewRequest(
		"PUT",
		"https://api.digitalocean.com/v2/domains/"+domain+"/records/"+strconv.Itoa(recordID),
		bytes.NewBuffer(jsonPayload),
	)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	os.Stdout.WriteString(string(body) + "\n")
	return nil
}

func ParseConfig() (*Config, error) {
	viper.SetConfigName("domains")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	config := &Config{}
	err = viper.Unmarshal(config)
	return config, err
}
