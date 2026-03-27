package helpers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type Containers struct {
	mongoContainer  *mongodb.MongoDBContainer
	rabbitContainer *rabbitmq.RabbitMQContainer
	redisContainer  testcontainers.Container

	MongoHost  string
	RabbitHost string
	RabbitPort int
	RedisHost  string
	RedisPort  int
}

func StartContainers(ctx context.Context) (*Containers, error) {
	mongoContainer, mongoHost, err := startMongoContainer(ctx)
	if err != nil {
		return nil, err
	}

	rabbitContainer, rabbitHost, rabbitPort, err := startRabbitContainer(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(mongoContainer)
		return nil, err
	}

	redisContainer, redisHost, redisPort, err := startRedisContainer(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(rabbitContainer)
		_ = testcontainers.TerminateContainer(mongoContainer)
		return nil, err
	}

	return &Containers{
		mongoContainer:  mongoContainer,
		rabbitContainer: rabbitContainer,
		redisContainer:  redisContainer,
		MongoHost:       mongoHost,
		RabbitHost:      rabbitHost,
		RabbitPort:      rabbitPort,
		RedisHost:       redisHost,
		RedisPort:       redisPort,
	}, nil
}

func (c *Containers) Close(ctx context.Context) error {
	var closeErr error

	if c.redisContainer != nil {
		if err := testcontainers.TerminateContainer(c.redisContainer); err != nil {
			closeErr = fmt.Errorf("terminate redis container: %w", err)
		}
	}
	if c.rabbitContainer != nil {
		if err := testcontainers.TerminateContainer(c.rabbitContainer); err != nil {
			if closeErr != nil {
				closeErr = fmt.Errorf("%v; terminate rabbitmq container: %w", closeErr, err)
			} else {
				closeErr = fmt.Errorf("terminate rabbitmq container: %w", err)
			}
		}
	}
	if c.mongoContainer != nil {
		if err := testcontainers.TerminateContainer(c.mongoContainer); err != nil {
			if closeErr != nil {
				closeErr = fmt.Errorf("%v; terminate mongo container: %w", closeErr, err)
			} else {
				closeErr = fmt.Errorf("terminate mongo container: %w", err)
			}
		}
	}

	_ = ctx
	return closeErr
}

func startMongoContainer(ctx context.Context) (container *mongodb.MongoDBContainer, host string, err error) {
	container, err = mongodb.Run(
		ctx,
		"mongo:latest",
		mongodb.WithUsername("test_user"),
		mongodb.WithPassword("test_pass"),
		mongodb.WithReplicaSet("rs0"),
	)
	if err != nil {
		return nil, "", fmt.Errorf("start mongo container: %w", err)
	}

	hostName, err := container.Host(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("resolve mongo host: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, "27017/tcp")
	if err != nil {
		return nil, "", fmt.Errorf("resolve mongo port: %w", err)
	}

	host = fmt.Sprintf("%s:%d/?directConnection=true", hostName, mappedPort.Int())

	if err := waitForMongoAuth(ctx, host); err != nil {
		return nil, "", err
	}

	return container, host, nil
}

func waitForMongoAuth(ctx context.Context, host string) error {
	uri := fmt.Sprintf("mongodb://test_user:test_pass@%s", host)
	deadline := time.Now().Add(30 * time.Second)
	for {
		client, err := mongo.Connect(options.Client().ApplyURI(uri).SetConnectTimeout(2 * time.Second))
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			pingErr := client.Ping(pingCtx, readpref.Primary())
			cancel()
			_ = client.Disconnect(context.Background())
			if pingErr == nil {
				return nil
			}
			err = pingErr
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("wait for mongo auth readiness: %w", err)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for mongo auth readiness: %w", ctx.Err())
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func startRabbitContainer(ctx context.Context) (container *rabbitmq.RabbitMQContainer, host string, port int, err error) {
	restoreTmpDir, err := useProjectTempDir()
	if err != nil {
		return nil, "", 0, fmt.Errorf("prepare rabbitmq temp dir: %w", err)
	}
	defer restoreTmpDir()

	container, err = rabbitmq.Run(
		ctx,
		"rabbitmq:3.13",
		rabbitmq.WithAdminUsername("test_user"),
		rabbitmq.WithAdminPassword("test_pass"),
	)
	if err != nil {
		return nil, "", 0, fmt.Errorf("start rabbitmq container: %w", err)
	}

	host, err = container.Host(ctx)
	if err != nil {
		return nil, "", 0, fmt.Errorf("resolve rabbitmq host: %w", err)
	}
	mappedPort, err := container.MappedPort(ctx, "5672/tcp")
	if err != nil {
		return nil, "", 0, fmt.Errorf("resolve rabbitmq port: %w", err)
	}

	return container, host, mappedPort.Int(), nil
}

func useProjectTempDir() (func(), error) {
	tempRoot, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	tempRoot = filepath.Join(tempRoot, "guestmanager-testcontainers")

	if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp(tempRoot, "rabbitmq-*")
	if err != nil {
		return nil, err
	}

	previousTmpDir, hadTmpDir := os.LookupEnv("TMPDIR")
	if err := os.Setenv("TMPDIR", tempDir); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, err
	}

	return func() {
		if hadTmpDir {
			_ = os.Setenv("TMPDIR", previousTmpDir)
		} else {
			_ = os.Unsetenv("TMPDIR")
		}
		_ = os.RemoveAll(filepath.Clean(tempDir))
	}, nil
}

func startRedisContainer(ctx context.Context) (testcontainers.Container, string, int, error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7.2-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp"),
		},
		Started: true,
	})
	if err != nil {
		return nil, "", 0, fmt.Errorf("start redis container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, "", 0, fmt.Errorf("resolve redis host: %w", err)
	}
	mappedPort, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		return nil, "", 0, fmt.Errorf("resolve redis port: %w", err)
	}

	return container, host, mappedPort.Int(), nil
}
