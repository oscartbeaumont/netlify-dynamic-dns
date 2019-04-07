# Netlify Dynamic DNS

[![Go Report Card](https://goreportcard.com/badge/github.com/oscartbeaumont/netlify-dynamic-dns)](https://goreportcard.com/report/github.com/oscartbeaumont/netlify-dynamic-dns)

A dynamic DNS client for [Netlify's Managed DNS](https://www.netlify.com/docs/dns/) service. It is a simple command line tool that retrieves your public IP and sets it to a DNS record using the Netlify API.

## Installation

[Click Here](https://github.com/oscartbeaumont/netlify-dynamic-dns/releases) to go to the Github Releases and download the correct installer for your platform. Alternatively, it can be run using Docker.

```bash
docker run oscartbeaumont/netlify-dynamic-dns:latest updater --access-token={Personal Access Token}
```

## Usage

Netlify Dynamic DNS has two modes it can be used in. When calling the binary use the following format.

```bash
netlify-ddns [MODE] [FLAGS]
```

### Update Mode

In update mode, the dynamic DNS will run once and then exit. This is good for use with a cron job or when you just want to manually update the DNS record.

```bash
netlify-ddns update --access-token={Personal Access Token}
```

### Updater Mode

In updater mode, the dynamic DNS will automatically update on an interval. This will continue until the application is terminated. This is good for running in a daemon.

```bash
netlify-ddns updater --access-token={Personal Access Token}
```

### Flags

| Flag         | Default     | Description                                                                                                              |
| ------------ | ----------- | ------------------------------------------------------------------------------------------------------------------------ |
| access-token |             | The personal access tokens for your Netlify accounts. Can be created in 'User Settings > Applications' on the dashboard. |
| domain       | example.com | The full domain for the DNS record                                                                                       |
| subdomain    | home        | The subdomain segment for the DNS record.                                                                                |
| ipv6         | true        | Whether the IPv6 'AAAA' DNS record should be updated.                                                                    |
| interval     | 5           | The interval (in minutes) to update your DNS record in the updater mode.                                                 |

### Netlify Access Token

To use this application you must first get an access token from Netlify. This allows the application to talk to the Netlify API on behalf of your account. To do that please go to the [Netlify OAuth applications](https://app.netlify.com/account/applications) page and create a new `Personal access token`.

## How It Works

Netlify Dynamic DNS uses [ident.me](https://ident.me) to determine the public IPv4 and IPv6 of the server and then talks to the Netlify API to create an A and AAAA record for those addresses. If duplicate DNS entries are detected they will be removed.

## Contributing

To work on the codebase, use the following commands.

```bash
git clone https://github.com/oscartbeaumont/netlify-dynamic-dns.git
go run ./cmd
```
