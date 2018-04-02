# zone.ee DNS A record updater

[![Go Report Card](https://goreportcard.com/badge/github.com/mannatera/dzdyndns)](https://goreportcard.com/report/github.com/mannatera/dzdyndns)

Update existing DNS A record using [ZoneID API](https://api.zone.eu/v2) version 2

## Usage

Clone the repo, build the program and run it. There are two possible ways to configure it:

* Through config.json
* Passing attributes via command line

To use config file just copy `config.json.dist` to `config.json` and fill in the missing property values.

When using command line attributes run the program as follows:

```shell
dzdyndns -fqdn <string> -token <string> -zoneid <string>
```

Property/attribute descriptions:

Property | Description
-------- | -----------
fqdn | Fully Qualified Domain Name that you wish to update (e.g. data.zone.ee)
token | API key for your ZoneID (can be generated under ZoneID account management)
zoneid | Your ZoneID username

## API keys

You can find info on how to create your API key in the [zone.ee help section](https://help.zone.eu/en/Knowledgebase/Article/View/546/0/zoneid-api-v2)

## Notes

This only updates existing A records. This means that when the specified FQDN does not exist it will not create it and does not update anything.