package defaults

import "time"

const (
	MaximumGithubEventsPerPage = 100
	PrometheusNamespace        = "tw"
	PrometheusSubsystem        = "github"
)

var (
	KafkaBroker            = "127.0.0.1:9092"
	KafkaProducerTopicName = "rollups"

	GithubEventsPerPage = MaximumGithubEventsPerPage

	StartupInfoReported = false

	SlidingWindowDuration = time.Minute * time.Duration(5)
	ProgramStartedAt      = time.Now()

	ServerBindHostPort = ":8080"
)
