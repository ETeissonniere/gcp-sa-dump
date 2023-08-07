# `gcp-sa-dump`

A simple script to list every service account along with its keys on GCP.

This script assumes you have full view access to every project on GCP, if you don't it may fail.

It will fetch on every project you may have access to every service account and the following metadata associted to them:

- account ID
- display name
- email
- state (whether it is disable or not)
- keys

## Usage

You may modulate the output format with the `-out` CLI flag. The following formats are supported:
- `text` (default)
- `csv`

### Text Output

```bash
$ go run .
```


### CSV Output
```bash
$ go run . -out csv
$ go run . -out csv > ./output.csv # to save to a file
```