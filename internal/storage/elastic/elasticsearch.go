package elastic

import (
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
)

const CourseIndex = "courses"

func NewElasticClient(password string, hosts []string) (*elasticsearch.Client, error) {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: hosts,
		Username:  "elastic",
		Password:  password,
	})
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("elastic: cannot connect to cluster: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("elastic: cluster returned error: %s", res.String())
	}
	return client, nil
}
