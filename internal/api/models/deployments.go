package models

// insights around tracking builds stats such as failed deployment stats

type Deployments struct {
	ID          string
	Project     string     `json:"project"`
	Environment string     `json:"environment"`
	Deployment  Deployment `json:"deployment"`
}

type Deployment struct {
	ID       string
	Build_ID int    `json:"build_id"`
	Project  string `json:"project"`
	Status   string `json:"status"`
}

type DeploymentMetrics struct {
	Project          string
	NumberOfBuilds   int
	FailedBuilds     int
	SuccessfulBuilds int
	BuildsPerMonth   int
}
