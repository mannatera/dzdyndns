package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	fqdn        string
	zoneid      string
	token       string
	host        string
	ip          string
	basePath    string
	apiVersion  = "2"
	url         = "https://api.zone.eu/v" + apiVersion + "/dns/"
	currentFile = "current.json"
)

// Config represents structure of config file
type Config struct {
	FQDN   string `json:"fqdn"`
	ZoneID string `json:"zoneid"`
	Token  string `json:"token"`
}

// Current represents structure of last update values
type Current struct {
	FQDN    string    `json:"fqdn"`
	IP      string    `json:"ip"`
	Updated time.Time `json:"updated"`
}

// ARecord defines A record structure defined in DataZone public API documentation
type ARecord struct {
	ID          string `json:"id"`
	AllowModify bool   `json:"modify"`
	AllowDelete bool   `json:"delete"`
	Destination string `json:"destination"`
	Resource    string `json:"resource_url"`
	Name        string `json:"name"`
}

func main() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		dir = "."
	}
	basePath = dir + string(filepath.Separator)

	config, err := ioutil.ReadFile(basePath + "config.json")
	if err == nil {
		conf := Config{}
		err = json.Unmarshal(config, &conf)
		if err == nil {
			fqdn = conf.FQDN
			token = conf.Token
			zoneid = conf.ZoneID
		}
	}

	flag.StringVar(&fqdn, "fqdn", fqdn, "Fully qualified domain name you want to update (e.g. api.zone.ee where api is the hostname and zone.ee is the domain)")
	flag.StringVar(&token, "token", token, "API access token")
	flag.StringVar(&zoneid, "zoneid", zoneid, "Your ZoneID")
	flag.Parse()

	if fqdn == "" || token == "" || zoneid == "" {
		log.Fatal("At least one of the required flag (-fqdn, -token, -zoneid) is missing! Append flag -h to get help for this command")
	}

	domain := fqdn
	if strings.Count(fqdn, ".") > 1 {
		fqdnParts := strings.SplitN(fqdn, ".", 2)
		host = fqdnParts[0]
		domain = fqdnParts[1]
	}

	url += domain + "/"

	ip = externalIP()

	currentFileContent, err := ioutil.ReadFile(basePath + currentFile)
	if err == nil {
		current := Current{}
		err = json.Unmarshal(currentFileContent, &current)
		if err == nil && current.IP == ip && current.FQDN == fqdn {
			fmt.Println("IP already matches for FQDN: " + fqdn + " (" + ip + "). No update needed.")
			os.Exit(0)
		}
	}

	records := aRecords()
	record := ARecord{}
	for _, v := range records {
		if v.Name == fqdn {
			record = v
			break
		}
	}

	if record.Destination == ip {
		fmt.Println("IP already matches for FQDN: " + fqdn + " (" + ip + "). No update needed.")
		updateCurrentFile()
		os.Exit(0)
	}

	record.Destination = ip // Set new ip
	updateARecord(record)
}

func aRecords() []ARecord {
	resp := requestHandler("GET", "a", nil)
	records := make([]ARecord, 0)

	body, _ := ioutil.ReadAll(resp.Body)

	err := json.Unmarshal(body, &records)
	if err != nil {
		log.Fatal(err)
	}

	return records
}

func updateARecord(r ARecord) {
	path := "a"
	if r.ID != "" {
		path += "/" + r.ID
	}

	resp := requestHandler("PUT", path, r)

	if resp.StatusCode == 404 {
		log.Fatal("Unable to update DNS A record: " + resp.Header.Get("X-Status-Message"))
	}

	fmt.Println(fqdn + " was successfully set to " + ip)
	updateCurrentFile()
}

func updateCurrentFile() {
	current := Current{fqdn, ip, time.Now().UTC()}
	currentFileContent, err := json.Marshal(current)
	if err != nil {
		fmt.Printf("Error when trying to create current file content: %s", err)
	} else {
		ioutil.WriteFile(basePath+currentFile, currentFileContent, 0660)
	}
}

func externalIP() string {
	resp, err := http.Get("http://checkip.amazonaws.com")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return string(bytes.TrimSpace(body))
}

func requestHandler(method, path string, body interface{}) *http.Response {
	content, err := json.Marshal(body)

	req, err := http.NewRequest(method, url+path, bytes.NewBuffer(content))
	if err != nil {
		log.Fatal(err)
	}

	authb64 := base64.StdEncoding.EncodeToString([]byte(zoneid + ":" + token))
	req.Header.Set("Authorization", "Basic "+authb64)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode == 401 {
		log.Fatal("Invalid API token, unauthorized")
	}

	return resp
}
