# Changelog

## 1.0.0

Initial release.  Provides basic data source functionality of fetching saved/scheduled search results from Cribl.

## 1.1.0

- Add support for using dashboard variables in Cribl Search queries.
- Add support for Grafana alerting.

## 1.1.1

- Remove unnecessary logic from CheckHealth
- Properly close HTTP response body in all cases

## 1.1.2

- Query timeout is now configurable.
- Add info & doc link to the timeout error message to encourage the user to leverage scheduled search.
- Upon timeout, cancel the query on the Cribl Search side.
- Allow saved search ID to be composed using dashboard `${variables}`.

## 1.1.3

- Fix "time" field data type.


## 1.1.4

- Add cancel logic to the backend
- Add cribl gov endpoints

## 1.1.5

- Fix bug: OAuth for cribl gov wasn't working properly.
- Fix bug: cloud org URL detection failed when there was a trailing slash (without any other path).
- Add a pre-provisioned sample dashboard.
- Better differentiation of plugin activity vs. regular Cribl UI activity.
- Enable local plugin development when using self-signed certs.
