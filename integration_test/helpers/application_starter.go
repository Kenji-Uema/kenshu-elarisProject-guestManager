package helpers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type ApplicationConfig struct {
	AppPort          int
	ClockHost        string
	ClockPort        int
	MongoHost        string
	MongoDatabase    string
	RabbitHost       string
	RabbitPort       int
	RedisHost        string
	RedisPort        int
	CleaningExchange string
	DayExchange      string
	HourExchange     string
}

func ApplicationStart(cfg ApplicationConfig) (stop func(), runErr <-chan error) {
	runCtx, cancelRun := context.WithCancel(context.Background())
	runErrCh := make(chan error, 1)
	waitDone := make(chan struct{})
	stopRequested := make(chan struct{})
	var stopOnce sync.Once

	binPath, cleanupBuild, err := buildApplicationBinary()
	if err != nil {
		runErrCh <- err
		return func() {}, runErrCh
	}
	stop = func() {
		stopOnce.Do(func() {
			close(stopRequested)
			cancelRun()
			_ = cleanupBuild()
		})
	}

	cmd := exec.Command(binPath)
	cmd.Dir = "/home/kenjiuema/Documents/projects/guestManager"
	cmd.Env = envWithOverrides(os.Environ(), applicationEnv(cfg))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		_ = cleanupBuild()
		runErrCh <- err
		return stop, runErrCh
	}

	go func() {
		defer close(waitDone)
		defer func() {
			_ = cleanupBuild()
		}()
		err := cmd.Wait()
		select {
		case <-stopRequested:
			if err == nil || isExpectedStopError(err) {
				runErrCh <- nil
				return
			}
		default:
		}
		if runCtx.Err() != nil && isExpectedStopError(err) {
			runErrCh <- nil
			return
		}
		runErrCh <- err
	}()

	stop = func() {
		stopOnce.Do(func() {
			close(stopRequested)
			cancelRun()

			if cmd.Process == nil {
				_ = cleanupBuild()
				return
			}

			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)

			go func() {
				timer := time.NewTimer(5 * time.Second)
				defer timer.Stop()

				select {
				case <-waitDone:
				case <-timer.C:
					_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				}
			}()
		})
	}

	return stop, runErrCh
}

func buildApplicationBinary() (string, func() error, error) {
	tempDir, err := os.MkdirTemp("", "guestmanager-integration-*")
	if err != nil {
		return "", func() error { return nil }, err
	}

	binPath := filepath.Join(tempDir, "guest-manager-under-test")
	buildCmd := exec.Command("go", "build", "-o", binPath, "./internal")
	buildCmd.Dir = "/home/kenjiuema/Documents/projects/guestManager"
	if err := buildCmd.Run(); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", func() error { return nil }, err
	}

	cleanup := func() error {
		return os.RemoveAll(tempDir)
	}

	return binPath, cleanup, nil
}

func isExpectedStopError(err error) bool {
	if err == nil {
		return false
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}

	return exitErr.ExitCode() < 0
}

func applicationEnv(cfg ApplicationConfig) map[string]string {
	return map[string]string{
		"SERVICE_NAME":                      "guest-manager",
		"VERSION":                           "integration-test",
		"SERVICE_HOST":                      "127.0.0.1",
		"SERVICE_PORT":                      fmt.Sprintf("%d", cfg.AppPort),
		"CLOCK_EMU_GRPC_HOST":               cfg.ClockHost,
		"CLOCK_EMU_GRPC_PORT":               fmt.Sprintf("%d", cfg.ClockPort),
		"MONGO_INITDB_ROOT_USERNAME":        "test_user",
		"MONGO_INITDB_ROOT_PASSWORD":        "test_pass",
		"MONGO_HOST":                        cfg.MongoHost,
		"MONGO_DATABASE":                    cfg.MongoDatabase,
		"GUEST_COLLECTION":                  "guest",
		"COTTAGE_COLLECTION":                "cottage",
		"BOOKING_COLLECTION":                "booking",
		"RABBITMQ_USERNAME":                 "test_user",
		"RABBITMQ_PASSWORD":                 "test_pass",
		"RABBITMQ_HOST":                     cfg.RabbitHost,
		"RABBITMQ_PORT":                     fmt.Sprintf("%d", cfg.RabbitPort),
		"CLEANING_EXCHANGE_NAME":            cfg.CleaningExchange,
		"CLEANING_EXCHANGE_KIND":            "direct",
		"CLEANING_EXCHANGE_DURABLE":         "false",
		"CLEANING_EXCHANGE_AUTO_DELETE":     "false",
		"CLEANING_PUBLISH_MANDATORY":        "false",
		"CLEANING_PUBLISH_IMMEDIATE":        "false",
		"HOUR_CHANGE_QUEUE_NAME":            fmt.Sprintf("integration.hour.queue.%d", cfg.AppPort),
		"HOUR_CHANGE_QUEUE_DURABLE":         "false",
		"HOUR_CHANGE_QUEUE_AUTO_DELETE":     "true",
		"HOUR_CHANGE_BINDING_EXCHANGE_NAME": cfg.HourExchange,
		"HOUR_CHANGE_BINDING_ROUTING_KEY":   "hour.change",
		"HOUR_CHANGE_CONSUME_CONSUMER":      fmt.Sprintf("integration-hour-consumer-%d", cfg.AppPort),
		"DAY_CHANGE_QUEUE_NAME":             fmt.Sprintf("integration.day.queue.%d", cfg.AppPort),
		"DAY_CHANGE_QUEUE_DURABLE":          "false",
		"DAY_CHANGE_QUEUE_AUTO_DELETE":      "true",
		"DAY_CHANGE_BINDING_EXCHANGE_NAME":  cfg.DayExchange,
		"DAY_CHANGE_BINDING_ROUTING_KEY":    "day.change",
		"DAY_CHANGE_CONSUME_CONSUMER":       fmt.Sprintf("integration-day-consumer-%d", cfg.AppPort),
		"REDIS_HOST":                        cfg.RedisHost,
		"REDIS_PORT":                        fmt.Sprintf("%d", cfg.RedisPort),
		"OTEL_EXPORTER_OTLP_ENDPOINT":       "127.0.0.1",
		"OTEL_EXPORTER_OTLP_GRPC_PORT":      "4317",
		"OTEL_EXPORTER_OTLP_HEALTH_PORT":    "13133",
		"OTEL_EXPORTER_OTLP_INSECURE":       "true",
	}
}

func envWithOverrides(baseEnv []string, overrides map[string]string) []string {
	result := make([]string, 0, len(baseEnv)+len(overrides))
	seen := make(map[string]struct{}, len(baseEnv))

	for _, entry := range baseEnv {
		key, _, ok := stringsCut(entry)
		if !ok {
			result = append(result, entry)
			continue
		}

		if override, ok := overrides[key]; ok {
			result = append(result, fmt.Sprintf("%s=%s", key, override))
		} else {
			result = append(result, entry)
		}
		seen[key] = struct{}{}
	}

	for key, value := range overrides {
		if _, ok := seen[key]; ok {
			continue
		}
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}

	return result
}

func stringsCut(entry string) (string, string, bool) {
	for i := 0; i < len(entry); i++ {
		if entry[i] == '=' {
			return entry[:i], entry[i+1:], true
		}
	}
	return "", "", false
}
