package main

import (
	"database/sql"
	"fmt"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/namsral/flag"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/umutcomlekci/mssql_exporter/collector"

	_ "github.com/denisenkom/go-mssqldb"
)

var (
	log = logrus.New()
)

type config struct {
	sqlServer		string
	sqlServerPort   int
	sqlUserName		string
	sqlPassword		string
}

func readAndValidateConfig() config {
	var result config

	flag.StringVar(&result.sqlServer, "sqlserver", "", "Sql Server")
	flag.IntVar(&result.sqlServerPort, "sqlport", 1433, "Sql Server Port")
	flag.StringVar(&result.sqlUserName, "sqlusername", "", "Sql Username")
	flag.StringVar(&result.sqlPassword, "sqlpassword", "", "Sql Password")

	flag.Parse()
	return result
}

func configureRoutes(app *fiber.App) {
	var landingPage = []byte(`<html>
		<head><title>MSSQL exporter for Prometheus</title></head>
		<body>
		<h1>MSSQL exporter for Prometheus</h1>
		<p><a href='/metrics'>Metrics</a></p>
		</body>
		</html>
		`)

	app.Get("/", func(c *fiber.Ctx) error {
		_, err := c.Write(landingPage)
		return err
	})
	
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
}

func main()  {

	config := readAndValidateConfig()
	if config.sqlServer == "" {
		log.Fatal("Missing SERVER information")
		return
	} else if config.sqlUserName == "" {
		log.Fatal("Missing USERNAME information")
		return
	} else if config.sqlPassword == "" {
		log.Fatal("Missing PASSWORD information")
		return
	}
	
	log.Infof("Connecting to server %s", config.sqlServer)
	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;", config.sqlServer, config.sqlUserName, config.sqlPassword, config.sqlServerPort)
	conn, err := sql.Open("mssql", connString)
	if err != nil {
		log.Fatal("Sql connection failed")
		return
	}
	defer conn.Close()

	coll := collector.New(conn, log)
	prometheus.MustRegister(coll)
	app := fiber.New()
	configureRoutes(app)
	log.Fatal(app.Listen(":8080"))
}