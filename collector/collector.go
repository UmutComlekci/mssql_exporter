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
	db  *sql.DB
	log *logrus.Logger

	up *prometheus.Desc

	mssqlInstanceLocalTime *prometheus.Desc
}

// New is the constructor method of Collector
func New(db *sql.DB, log *logrus.Logger) *Collector {
	return &Collector{
		db:  db,
		log: log,
		up:  prometheus.NewDesc("mssql_up", "Whether the MSSQL scrape was successful", nil, nil),

		mssqlInstanceLocalTime: prometheus.NewDesc("mssql_instance_local_time", "Number of seconds since epoch on local instance", nil, nil),
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector to the provided channel and returns once
// the last descriptor has been sent.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up

	ch <- c.mssqlInstanceLocalTime
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

		c.log.Info("Scrape completed")
	}
}

func collectLocalTime(c *Collector, ch chan<- prometheus.Metric)  {
	rows := c.db.QueryRow("SELECT DATEDIFF(second, '19700101', GETUTCDATE())")
	var localTime float64
	err := rows.Scan(&localTime)
	if err != nil {
		c.log.Fatal("LocalTine scan failed:", err.Error())
	}
	ch <- prometheus.MustNewConstMetric(c.mssqlInstanceLocalTime, prometheus.GaugeValue, localTime)
}
