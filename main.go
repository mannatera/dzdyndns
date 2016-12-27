package main

import (
	"bytes"
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

	"github.com/google/go-querystring/query"
)

var (
	fqdn        string
	zoneid      string
	token       string
	host        string
	domain      string
	ip          string
	basePath    string
	apiVersion  = "1"
	url         = "https://api.zone.eu/v" + apiVersion + "/domains/"
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

// Response is the response type defined in DataZone public API documentation
type Response struct {
	Status   int               `json:"status"`
	Messages []string          `json:"messages"`
	Params   map[string]Domain `json:"params"`
}

// Domain defines domain response param structure defined in DataZone public API documentation
type Domain struct {
	NszoneID       string `json:"nszone_id"`
	Adomain        string
	AdomainUnicode string `json:"adomain_unicode"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	TTL            string
	A              map[string]ARecord
}

// ARecord defines A record structure defined in DataZone public API documentation
type ARecord struct {
	ID          string
	AllowModify bool   `json:"allow_modify"`
	AllowDelete bool   `json:"allow_delete"`
	Content     string `json:"content"`
	Host        string
}

// ARecordPost structure of POST message for updating or creating A records, defined in DataZone public API documentation
type ARecordPost struct {
	Type    string `url:"type"`    // Always with value A
	Prefix  string `url:"prefix"`  // Value from the host variable
	Content string `url:"content"` // External IP of the client
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

	if strings.Count(fqdn, ".") > 1 {
		fqdnParts := strings.SplitN(fqdn, ".", 2)
		host = fqdnParts[0]
		domain = fqdnParts[1]
	} else {
		domain = fqdn
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
		if v.Host == fqdn {
			record = v
			break
		}
	}
	updateARecord(record)
}

func aRecords() map[string]ARecord {
	resp := requestHandler("GET", "records", nil)
	response := Response{}

	err := json.Unmarshal(resp, &response)
	if err != nil {
		log.Fatal(err)
	}

	return response.Params[domain].A
}

func updateARecord(r ARecord) {
	if r.Content == ip {
		fmt.Println("IP already matches for FQDN: " + fqdn + " (" + ip + "). No update needed.")
		updateCurrentFile()
		os.Exit(0)
	}
	path := "records"
	if r.ID != "" {
		path += "/" + r.ID
	}
	aRecord := ARecordPost{"A", host, ip}
	requestHandler("POST", path, aRecord)
	fmt.Println(fqdn + " was successfully set to " + ip)
	updateCurrentFile()
}

func updateCurrentFile() {
	current := Current{fqdn, ip, time.Now().UTC()}
	currentFileContent, err := json.Marshal(current)
	if err != nil {
		fmt.Printf("Error when trying to create current file content: %s", err)
	} else {
		ioutil.WriteFile(basePath+currentFile, currentFileContent, 600)
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

func requestHandler(method, path string, body interface{}) []byte {
	content := ""
	if body != nil {
		v, _ := query.Values(body)
		content = v.Encode()
	}

	req, err := http.NewRequest(method, url+path, strings.NewReader(content))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("X-ZoneID-Token", zoneid+":"+token)
	req.Header.Set("X-ResponseType", "JSON")

	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	if string(bodyContent) == "invalid api token" {
		log.Fatal("Invalid API token")
	}

	return bodyContent
}
