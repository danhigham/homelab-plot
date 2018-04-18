package bosh

import (
	"github.com/cloudfoundry-community/gogobosh"
)

func GetDeployments(config *gogobosh.Config) ([]gogobosh.Deployment, error) {
	c, err := gogobosh.NewClient(config)
	if err != nil {
		return nil, err
	}
	return c.GetDeployments()
}

func GetDeploymentVMs(config *gogobosh.Config, deploymentName string) ([]gogobosh.VM, error) {

	c, err := gogobosh.NewClient(config)
	if err != nil {
		return nil, err
	}
	return c.GetDeploymentVMs(deploymentName)
}
