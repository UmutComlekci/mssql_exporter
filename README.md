# MSSQL Prometheus exporter
Prometheus exporter for MSSQL server metrics.

## Installation

### From source

You need to have a Go 1.15+ environment configured. Clone the repo (outside your `GOPATH`) and then:

```bash
go build -o mssql_exporter
./mssql_exporter --sqlserver=[YOUR CONNECTION STRING] --sqlusername=[YOUR USER NAME] --sqlpassword=[YOUR PASSWORD]
```

### Using Docker

```bash
docker image build -t umutcomlekci/mssql_exporter -f Dockerfile .
docker run -d -p 8080:8080 -e SQLSERVER=[YOUR CONNECTION STRING] -e SQLUSERNAME=[YOUR USER NAME] -e SQLPASSWORD=[YOUR PASSWORD] umutcomlekci/mssql_exporter
```