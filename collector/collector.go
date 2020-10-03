package collector

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type collectorFunc func(c *Collector, m *metric, ch chan<- prometheus.Metric)

type metric struct {
	describer *prometheus.Desc
	query     string
	collector collectorFunc
}

// Collector is the interface implemented by anything that can be used by
// Prometheus to collect metrics. A Collector has to be registered for
// collection.
type Collector struct {
	db  *sql.DB
	log *logrus.Logger

	up      *prometheus.Desc
	metrics []metric
}

// New is the constructor method of Collector
func New(db *sql.DB, log *logrus.Logger) *Collector {
	return &Collector{
		db:  db,
		log: log,
		up:  prometheus.NewDesc("mssql_up", "Whether the MSSQL scrape was successful", nil, nil),
		metrics: []metric{
			{
				describer: prometheus.NewDesc("mssql_instance_local_time", "Number of seconds since epoch on local instance", nil, nil),
				collector: collectLocalTimeMetric,
				query:     "SELECT DATEDIFF(second, '19700101', GETUTCDATE())",
			},
			{
				describer: prometheus.NewDesc("mssql_connections", "Number of active connections", []string{"database", "state"}, nil),
				collector: collectConnectionsMetrics,
				query:     "SELECT DB_NAME(sP.dbid), COUNT(sP.spid) FROM sys.sysprocesses sP GROUP BY DB_NAME(sP.dbid)",
			},
			{
				describer: prometheus.NewDesc("mssql_deadlocks", "Number of lock requests per second that resulted in a deadlock since last restart", nil, nil),
				collector: collectDeadLocksMetric,
				query:     "SELECT cntr_value FROM sys.dm_os_performance_counters where counter_name = 'Number of Deadlocks/sec' AND instance_name = '_Total'",
			},
		},
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector to the provided channel and returns once
// the last descriptor has been sent.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up

	for _, metric := range c.metrics {
		ch <- metric.describer
	}
}

// Collect is called by the Prometheus registry when collecting metrics.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.log.Info("Running scrape")

	if err := c.db.Ping(); err != nil {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		c.log.WithError(err).Error("Error during scrape")
	} else {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)

		for _, metric := range c.metrics {
			metric.collector(c, &metric, ch)
		}

		c.log.Info("Scrape completed")
	}
}

func collectLocalTimeMetric(c *Collector, m *metric, ch chan<- prometheus.Metric) {
	rows := c.db.QueryRow(m.query)
	var localTime float64

	if err := rows.Scan(&localTime); err != nil {
		c.log.Fatal("LocalTine scan failed:", err.Error())
		return
	}

	ch <- prometheus.MustNewConstMetric(m.describer, prometheus.GaugeValue, localTime)
}

func collectConnectionsMetrics(c *Collector, m *metric, ch chan<- prometheus.Metric) {
	rows, err := c.db.Query(m.query)
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

		ch <- prometheus.MustNewConstMetric(m.describer, prometheus.GaugeValue, connections, dbName, "current")
	}
}

func collectDeadLocksMetric(c *Collector, m *metric, ch chan<- prometheus.Metric) {
	rows := c.db.QueryRow(m.query)
	var deadlocks float64

	if err := rows.Scan(&deadlocks); err != nil {
		c.log.Fatal("DeadLocks scan failed:", err.Error())
		return
	}
	ch <- prometheus.MustNewConstMetric(m.describer, prometheus.GaugeValue, deadlocks)
}
