package collector

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Collector is the interface implemented by anything that can be used by
// Prometheus to collect metrics. A Collector has to be registered for
// collection.
type Collector struct {
	db  					*sql.DB
	log 					*logrus.Logger

	up 						*prometheus.Desc
	mssqlInstanceLocalTime 	*prometheus.Desc
	mssqlConnections 		*prometheus.Desc
}

// New is the constructor method of Collector
func New(db *sql.DB, log *logrus.Logger) *Collector {
	return &Collector{
		db:  db,
		log: log,
		up:  prometheus.NewDesc("mssql_up", "Whether the MSSQL scrape was successful", nil, nil),

		mssqlInstanceLocalTime: prometheus.NewDesc("mssql_instance_local_time", "Number of seconds since epoch on local instance", nil, nil),
		mssqlConnections: prometheus.NewDesc("mssql_connections", "Number of active connections", []string { "database", "state" }, nil),
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector to the provided channel and returns once
// the last descriptor has been sent.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up

	ch <- c.mssqlInstanceLocalTime
	ch <- c.mssqlConnections
}

// Collect is called by the Prometheus registry when collecting metrics.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.log.Info("Running scrape")

	if err := c.db.Ping(); err != nil {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		c.log.WithError(err).Error("Error during scrape")
	} else {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)

		collectLocalTime(c, ch)
		collectConnections(c, ch)

		c.log.Info("Scrape completed")
	}
}

func collectLocalTime(c *Collector, ch chan<- prometheus.Metric) {
	rows := c.db.QueryRow("SELECT DATEDIFF(second, '19700101', GETUTCDATE())")
	var localTime float64
	
	if err := rows.Scan(&localTime); err != nil {
		c.log.Fatal("LocalTine scan failed:", err.Error())
		return
	}
	ch <- prometheus.MustNewConstMetric(c.mssqlInstanceLocalTime, prometheus.GaugeValue, localTime)
}

func collectConnections(c *Collector, ch chan<- prometheus.Metric) {
	rows, err := c.db.Query("SELECT DB_NAME(sP.dbid), COUNT(sP.spid) FROM sys.sysprocesses sP GROUP BY DB_NAME(sP.dbid)")
	if err != nil {
		c.log.Fatal("Collect connection failed:", err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		var dbName string
		var connections float64
		if err := rows.Scan(&dbName, &connections); err != nil {
			c.log.Fatal("Collect connection scan failed:", err.Error())
			return
		}

		ch <- prometheus.MustNewConstMetric(c.mssqlConnections, prometheus.GaugeValue, connections, dbName, "current")
	}
}