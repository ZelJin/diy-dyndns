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
)

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

func main() {
	token := os.Getenv("DO_TOKEN")
	domain := os.Getenv("DO_DOMAIN")
	subdomains := os.Getenv("DO_SUBDOMAINS")
	if token != "" && domain != "" && subdomains != "" {
		gocron.Every(10).Minutes().Do(CheckIPAddress)
		<-gocron.Start()
	} else {
		os.Stderr.WriteString("Env variables are not configured correctly.\n")
		os.Exit(1)
	}
}

// CheckIPAddress checks the IP address assigned to the domain,
// compares it to the real IP address of the server and
// orders to change the DNS record if needed.
func CheckIPAddress() {
	token := os.Getenv("DO_TOKEN")
	domain := os.Getenv("DO_DOMAIN")
	subdomains := strings.Split(os.Getenv("DO_SUBDOMAINS"), ",")
	externalIP, err := GetExternalIP()
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	os.Stdout.WriteString("External IP: " + externalIP + "\n")
	domainRecords, err := GetDomainRecords(domain, token)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	for _, record := range domainRecords {
		for _, subdomain := range subdomains {
			if record.Type == "A" && record.Name == subdomain {
				os.Stdout.WriteString(record.Name + " " + record.Data + "\n")
				if externalIP != record.Data {
					SetDomainRecord(domain, record.ID, externalIP, token)
				}
			}
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
