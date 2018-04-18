package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/cloudfoundry-community/gogobosh"
	"github.com/danhigham/homelab-plot/bosh"
	"github.com/danhigham/homelab-plot/performance"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

type HostsQuery struct {
	Hosts      []string `json:"hosts"`
	Datacenter string   `json:"dc"`
	Metrics    []string `json:"metrics"`
}

type VMsQuery struct {
	VMs        []string `json:"vms"`
	Datacenter string   `json:"dc"`
	Metrics    []string `json:"metrics"`
}

func main() {

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Static("./static"))

	// Routes
	e.POST("/findhosts.json", findHosts)
	e.POST("/hosts.json", hosts)
	e.GET("/deployments.json", deployments)
	e.GET("/deployment/:id/vms.json", deployment)
	e.POST("/vms.json", vms)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}

func boshConfig() *gogobosh.Config {
	clientConfig := &gogobosh.Config{
		BOSHAddress:       os.Getenv("BOSH_ADDRESS"),
		ClientID:          os.Getenv("BOSH_CLIENT"),
		ClientSecret:      os.Getenv("BOSH_CLIENT_SECRET"),
		UAAAuth:           true,
		HttpClient:        http.DefaultClient,
		SkipSslValidation: true,
	}
	return clientConfig
}

func deployments(c echo.Context) error {
	boshConfig := boshConfig()
	deployments, err := bosh.GetDeployments(boshConfig)

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%s", err))
	}

	return c.JSON(http.StatusOK, deployments)
}

func deployment(c echo.Context) error {

	boshConfig := boshConfig()
	vms, err := bosh.GetDeploymentVMs(boshConfig, c.Param("id"))

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%s", err))
	}

	return c.JSON(http.StatusOK, vms)
}

func findHosts(c echo.Context) error {

	q := new(HostsQuery)
	if err := c.Bind(q); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%s", err))
	}

	hostIPs, err := performance.FindHosts(os.Getenv("VCENTER_URI"), q.Datacenter)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%s", err))
	}
	return c.JSON(http.StatusOK, hostIPs)
}

func vms(c echo.Context) error {

	q := new(VMsQuery)
	if err := c.Bind(q); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%s", err))
	}
	vms, err := performance.VMs(os.Getenv("VCENTER_URI"),
		q.Datacenter,
		q.VMs,
		q.Metrics)

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%s", err))
	}

	return c.JSON(http.StatusOK, vms)
}

func hosts(c echo.Context) error {

	q := new(HostsQuery)
	if err := c.Bind(q); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%s", err))
	}
	hosts, err := performance.Hosts(os.Getenv("VCENTER_URI"),
		q.Datacenter,
		q.Hosts,
		q.Metrics)

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%s", err))
	}

	return c.JSON(http.StatusOK, hosts)
}
