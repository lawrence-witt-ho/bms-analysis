# bms-analysis

Desktop app for analysing BMS kibana logs

## Dependencies

- [Go](https://go.dev/doc/install)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation/)
- [Node](https://nodejs.org/en/download)

## Run

### Generate

If you have no raw logs saved you will need to connect to kibana and pull the latest records.

Activate the ho-it-live VPN before attempting to connect and include an `.env` file in the root of the repository with your LDAP credentials:

```
LDAP_USERNAME=...
LDAP_PASSWORD=...
```

Coordinate processing unfortunately takes a long time due to limitations in the current projection implementation (On^3). This could be improved in the future if needed but in the mean time it is advisable to avoid regenerating large datasets whenever possible.

#### Errors Data

Run `make errors`. This will pull all the kibana logs with `message` types in the known error list from the last month, then compute the similarity between their `errorMessage` properties.

#### Alerts Data

Run `make alerts`. This will pull all the kibana watcher executions from the last month that resulted in a successful fire, attempt to locate their associated log, then compute the similarity between the `errorMessage` properties of all these associated logs. Unfortunately, this is not all that useful, because many executions don't appear to show up in the slack channel at all while others appear in the channel but have duplicate executions.

### View

Once you have generated the data, or sourced pre-generated files, run `make dev` to start the application in dev mode. Select your generated coordinates file using the file input to see the graph.
