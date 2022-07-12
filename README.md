# todoistbackup

A backup syncronization tool for [Todoist](https://todoist.com).


## DESCRIPTION

todoistbackup downloads backup files from the
[Todoist Sync API](https://developer.todoist.com/sync/v8/) and uploads missing 
backup files to a user provided S3 bucket. The project is still immature, threfore use at your own risk.


## USAGE

The backup tool can be used with Docker as follows.

```bash
docker run \
  --name todoistbackup \
  --rm \
  --volume $(pwd)/config.json:/etc/todoistbackup/config.json \
  xinau/todoistbackup
```


## CONFIGURATION

The configuration is written in JSON and loaded using the `--config.file` flag. 
It provides the following attributes for configuring various aspects of the backup tool.

`client.timeout <int optional>` - HTTP client request timeout in seconds.

`client.token <string>` - API token obtained from the
[integrations page](https://todoist.com/app/settings/integrations).  

`store.bucket <string>` - S3 bucket name. If it doesn't exist it will be created.

`store.endpoint <string>` - S3 API endpoint to use.

`store.region <string optional>` - S3 region the bucket will be created in.

`store.access-key <string>` - S3 access key to use.

`store.secret-key <string>` - S3 secret key to use.

`store.insecure <bool>` - Set to `true` to allow insecure S3 connections.


## BUILD

The backup tool is developed in [Go](https://golang.org) and can be build with 
a recent toolchain.

```bash
git clone https://github.com/xinau/todoistbackup.git
cd todoistbackup
go build -o out/todoistbackup ./cmd/todoistbackup
```


## LICENSE

This project is under [MIT license](./LICENSE).
