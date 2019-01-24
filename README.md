# Netlify Dynamic DNS
A dynamic DNS client for [Netlify's Managed DNS](https://www.netlify.com/docs/dns/) service. Every 5 minutes (configurable) the application will (using [ident.me](https://ident.me)) determine the server's public IPv4 and IPv6 address and check to see if they match the address's currently in the Netlify A and AAAA DNS records. If they do not match the existing values the values will be update. If duplicate DNS entries are detected the duplicates will be removed.

## Access Token
To use this application you must first get an access token from Netlify. This allows the application to talk to the Netlify API on behalf of your account. To do that go to the [Netlify OAuth applications](https://app.netlify.com/account/applications) page and create a new "Personal access token".

## Using the Docker Container
You can easily deploy this application in a Docker container using the following command:
```bash
docker run -e DOMAIN=example.com -e HOST=home -e ACCESS_TOKEN={Personal Access Token} oscartbeaumont/netlify-dynamic-dns:latest
```
You will need to replace the `{Personal Access Token}` and the domain `example.com` with your access token and domain. With the options used in this example your public IP will be mapped to the domain `home.example.com`. Alternatively you can parse an `ACCESS_TOKEN_FILE` which is the path to a file on disk that contains the access token (this is great for Docker Swarm secrets).

## Using the Binary
Download The Binary From The [Github Releases](https://github.com/oscartbeaumont/netlify-dynamic-dns/releases) and use the following command:
```bash
./netlify-dynamic-dns -domain example.com -host home -access-token {Personal Access Token}
```
For more information & options about the argument you can use with the command run: `./netlify-dynamic-dns -help`. Alternatively you can parse an `-access-token-file` which is the path to a file on disk that contains the access token.

# How To Compile
Even though unnecessary in most cases you can compile the source by using the commands below.
To build the docker image you need [Docker](https://docker.com) installed on your system. To build the docker version use the following command:
```bash
docker build -t oscartbeaumont/netlify-dynamic-dns .
```
To compile the binary you need [Go Lang](https://golang.org) installed on your system. To compile the binary version use the following command:
```bash
env GOOS=linux GOARCH=amd64 go build -o netlify-dynamic-dns ./cmd
```
