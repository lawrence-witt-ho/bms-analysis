# bms-analysis

Desktop app for analysing BMS kibana logs

## Dependencies

- [Go](https://go.dev/doc/install)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation/)
- [Node](https://nodejs.org/en/download)

## Run

### Generate

To generate data, run `make run` from the repository root. You should activate the ho-it-live VPN before attempting to connect.

Include an `.env` file to authenticate with kibana if you have no raw logs saved.

```
LDAP_USERNAME=...
LDAP_PASSWORD=...
```

Coordinate processing unfortunately takes a long time due to limitations in the current projection implementation (On^3). This could be improved in the future if needed but in the mean time it is advisable to avoid regenerating large datasets whenever possible.

### View

Once you have generated the data, or placed pre-generated files in the repository root, run `wails dev` to start the application in dev mode.
