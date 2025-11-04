package main

import (
	"os"

	"github.com/criblcloud/search-datasource/pkg/plugin"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

func main() {
	// TODO: ideally we'd be able to glean this from package.json.
	// I played around with defining this at build time via ldflags, but that took us down
	// a path where if we modify build scripts, we'd deviate from easy upgradeability in the
	// future.  I have an open thread with Grafana community Slack to find a better way.
	plugin.SetVersion("1.1.4")

	// Start listening to requests sent from Grafana. This call is blocking so
	// it won't finish until Grafana shuts down the process or the plugin choose
	// to exit by itself using os.Exit. Manage automatically manages life cycle
	// of datasource instances. It accepts datasource instance factory as first
	// argument. This factory will be automatically called on incoming request
	// from Grafana to create different instances of SampleDatasource (per datasource
	// ID). When datasource configuration changed Dispose method will be called and
	// new datasource instance created using NewSampleDatasource factory.
	if err := datasource.Manage("criblcloud-search-datasource", plugin.NewDatasource, datasource.ManageOpts{}); err != nil {
		log.DefaultLogger.Error(err.Error())
		os.Exit(1)
	}
}
