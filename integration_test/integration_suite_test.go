package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/integration_test/helpers"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	projectRoot         = "/home/kenjiuema/Documents/projects/guestManager"
	guestsFixturePath   = projectRoot + "/test_data/guests_fixture.json"
	bookingsFixturePath = projectRoot + "/test_data/bookings_fixture.json"
	cottagesFixturePath = projectRoot + "/test_data/cottages_fixture.json"
	suiteDBName         = "test_db"
	suiteCleaningEx     = "ex.cleaning.request"
	suiteDayEx          = "ex.day_change.event"
	suiteHourEx         = "ex.hour_change.event"
)

var (
	suiteContainers *helpers.Containers
	suiteClock      *helpers.MutableClockServer
	suiteClockStop  func()
	suiteAppPort    int
	suiteStopMain   func()
	suiteRunErrCh   <-chan error

	suiteRawMongoClient    *mongo.Client
	suiteGuestCollection   *mongo.Collection
	suiteBookingCollection *mongo.Collection
	suiteCottageCollection *mongo.Collection
	suiteRedisClient       *redis.Client
	suiteRabbitConn        *amqp.Connection
	suiteRabbitChannel     *amqp.Channel
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	DeferCleanup(cancel)

	var err error
	suiteContainers, err = helpers.StartContainers(ctx)
	if err != nil {
		Fail(fmt.Sprintf("integration test setup failed: start containers: %v", err))
	}

	if err := seedFixtures(context.Background()); err != nil {
		Fail(fmt.Sprintf("integration test setup failed: seed fixtures: %v", err))
	}

	suiteRawMongoClient, err = mongo.Connect(options.Client().ApplyURI(fmt.Sprintf("mongodb://test_user:test_pass@%s", suiteContainers.MongoHost)))
	if err != nil {
		Fail(fmt.Sprintf("integration test setup failed: connect raw mongo client: %v", err))
	}

	db := suiteRawMongoClient.Database(suiteDBName)
	suiteGuestCollection = db.Collection("guest")
	suiteBookingCollection = db.Collection("booking")
	suiteCottageCollection = db.Collection("cottage")

	suiteRedisClient = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", suiteContainers.RedisHost, suiteContainers.RedisPort),
	})
	if err := suiteRedisClient.Ping(context.Background()).Err(); err != nil {
		Fail(fmt.Sprintf("integration test setup failed: ping redis: %v", err))
	}

	suiteRabbitConn, err = amqp.Dial(amqp.URI{
		Scheme:   "amqp",
		Username: "test_user",
		Password: "test_pass",
		Host:     suiteContainers.RabbitHost,
		Port:     suiteContainers.RabbitPort,
	}.String())
	if err != nil {
		Fail(fmt.Sprintf("integration test setup failed: dial rabbitmq: %v", err))
	}

	suiteRabbitChannel, err = suiteRabbitConn.Channel()
	if err != nil {
		Fail(fmt.Sprintf("integration test setup failed: open rabbitmq channel: %v", err))
	}

	Expect(suiteRabbitChannel.ExchangeDeclare(suiteCleaningEx, "direct", false, false, false, false, nil)).To(Succeed())
	Expect(suiteRabbitChannel.ExchangeDeclare(suiteDayEx, "direct", false, true, false, false, nil)).To(Succeed())
	Expect(suiteRabbitChannel.ExchangeDeclare(suiteHourEx, "direct", false, true, false, false, nil)).To(Succeed())

	initialNow := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	var clockHost string
	var clockPort int
	clockHost, clockPort, suiteClock, suiteClockStop, err = helpers.StartClockServer(initialNow)
	if err != nil {
		Fail(fmt.Sprintf("integration test setup failed: start clock server: %v", err))
	}

	suiteAppPort, err = helpers.FreeTCPPort()
	if err != nil {
		Fail(fmt.Sprintf("integration test setup failed: reserve app port: %v", err))
	}

	suiteStopMain, suiteRunErrCh = helpers.ApplicationStart(helpers.ApplicationConfig{
		AppPort:          suiteAppPort,
		ClockHost:        clockHost,
		ClockPort:        clockPort,
		MongoHost:        suiteContainers.MongoHost,
		MongoDatabase:    suiteDBName,
		RabbitHost:       suiteContainers.RabbitHost,
		RabbitPort:       suiteContainers.RabbitPort,
		RedisHost:        suiteContainers.RedisHost,
		RedisPort:        suiteContainers.RedisPort,
		CleaningExchange: suiteCleaningEx,
		DayExchange:      suiteDayEx,
		HourExchange:     suiteHourEx,
	})

	if err := waitForHTTP200OrExit(suiteAppPort, suiteRunErrCh, 30*time.Second); err != nil {
		if suiteStopMain != nil {
			suiteStopMain()
			suiteStopMain = nil
			suiteRunErrCh = nil
		}
		Fail(fmt.Sprintf("integration test setup failed: %v", err))
	}
})

var _ = AfterSuite(func() {
	if suiteStopMain != nil {
		suiteStopMain()
		select {
		case err := <-suiteRunErrCh:
			Expect(err).NotTo(HaveOccurred())
		case <-time.After(45 * time.Second):
			Fail("main did not stop after context cancel")
		}
	}

	if suiteRabbitChannel != nil {
		Expect(suiteRabbitChannel.Close()).To(Succeed())
	}
	if suiteRabbitConn != nil {
		Expect(suiteRabbitConn.Close()).To(Succeed())
	}
	if suiteRedisClient != nil {
		Expect(suiteRedisClient.Close()).To(Succeed())
	}
	if suiteRawMongoClient != nil {
		Expect(suiteRawMongoClient.Database(suiteDBName).Drop(context.Background())).To(Succeed())
		Expect(suiteRawMongoClient.Disconnect(context.Background())).To(Succeed())
	}
	if suiteClockStop != nil {
		suiteClockStop()
	}
	if suiteContainers != nil {
		Expect(suiteContainers.Close(context.Background())).To(Succeed())
	}
})

func resetFixtureState(t FullGinkgoTInterface) {
	t.Helper()

	Expect(seedFixtures(context.Background())).To(Succeed())
	Expect(suiteRedisClient.FlushDB(context.Background()).Err()).NotTo(HaveOccurred())

	suiteClock.Set(time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC))
}

func seedFixtures(ctx context.Context) error {
	return helpers.SeedMongoFromFixtures(ctx, suiteContainers.MongoHost, suiteDBName, map[string]string{
		"guest":   guestsFixturePath,
		"booking": bookingsFixturePath,
		"cottage": cottagesFixturePath,
	})
}

func waitForHTTP200OrExit(appPort int, runErrCh <-chan error, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("http://127.0.0.1:%d/readyz", appPort)
	client := &http.Client{Timeout: 2 * time.Second}

	for {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case runErr := <-runErrCh:
			if runErr != nil {
				return fmt.Errorf("application exited before ready: %w", runErr)
			}
			return fmt.Errorf("application exited before ready")
		default:
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s to return 200", url)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func cacheCheckInBookings(t FullGinkgoTInterface, keyDate time.Time, bookings []documents.Booking) {
	t.Helper()

	payload, err := json.Marshal(bookings)
	Expect(err).NotTo(HaveOccurred())

	key := fmt.Sprintf("checkin.%s", keyDate.UTC().Format("2006-01-02"))
	Expect(suiteRedisClient.Set(context.Background(), key, payload, 30*time.Minute).Err()).NotTo(HaveOccurred())
}

func mustObjectID(t FullGinkgoTInterface, hex string) bson.ObjectID {
	t.Helper()

	id, err := bson.ObjectIDFromHex(hex)
	Expect(err).NotTo(HaveOccurred())

	return id
}
