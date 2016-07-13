package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	etcd "github.com/coreos/etcd/client"
	"github.com/gogo/protobuf/proto"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/nsqio/go-nsq"
	"github.com/opsee/basic/schema"
	"github.com/opsee/cats/checks"
	"github.com/opsee/cats/checks/results"
	"github.com/opsee/cats/checks/worker"
	"github.com/opsee/cats/store"
	log "github.com/opsee/logrus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

var (
	checkResultsHandled = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "check_results_handled",
		Help: "Total number of check results processed.",
	})
)

func init() {
	prometheus.MustRegister(checkResultsHandled)
}

func main() {
	viper.SetEnvPrefix("cats")
	viper.AutomaticEnv()

	viper.SetDefault("log_level", "info")
	logLevelStr := viper.GetString("log_level")
	logLevel, err := log.ParseLevel(logLevelStr)
	if err != nil {
		log.WithError(err).Error("Could not parse log level, using default.")
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)

	go func() {
		hostname, err := os.Hostname()
		if err != nil {
			log.WithError(err).Error("Error getting hostname.")
			return
		}

		ticker := time.Tick(5 * time.Second)
		for {
			<-ticker
			err = prometheus.Push("pracovnik", hostname, "172.30.35.35:9091")
			if err != nil {
				log.WithError(err).Error("Error pushing to pushgateway.")
			}
		}
	}()

	nsqConfig := nsq.NewConfig()
	nsqConfig.MaxInFlight = 4

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	maxTasks := viper.GetInt("max_tasks")
	// in-memory cache of customerId -> bastionId
	bastionMap := map[string]string{}

	consumer, err := worker.NewConsumer(&worker.ConsumerConfig{
		Topic:            "_.results",
		Channel:          "dynamo-results-worker",
		LookupdAddresses: viper.GetStringSlice("nsqlookupd_addrs"),
		NSQConfig:        nsqConfig,
		HandlerCount:     maxTasks,
	})

	if err != nil {
		log.WithError(err).Fatal("Failed to create consumer.")
	}

	nsqdHost := viper.GetString("nsqd_host")
	producer, err := nsq.NewProducer(nsqdHost, nsqConfig)
	if err != nil {
		log.WithError(err).Fatal("Failed to create producer")
	}

	db, err := sqlx.Open("postgres", viper.GetString("postgres_conn"))
	if err != nil {
		log.WithError(err).Fatal("Cannot connect to database.")
	}

	// TODO(greg): All of the etcd stuff can go once bastions report their
	// bastion id in check results.
	etcdCfg := etcd.Config{
		Endpoints:               []string{viper.GetString("etcd_address")},
		Transport:               etcd.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}
	etcdClient, err := etcd.New(etcdCfg)
	if err != nil {
		log.WithError(err).Fatal("Cannot connect to etcd.")
	}

	kapi := etcd.NewKeysAPI(etcdClient)

	awsSession := session.New(&aws.Config{Region: aws.String("us-west-2")})
	dynamo := &results.DynamoStore{dynamodb.New(awsSession)}
	s3Store := &results.S3Store{
		S3Client:   s3.New(awsSession),
		BucketName: viper.GetString("results_s3_bucket"),
	}

	consumer.AddHandler(func(msg *nsq.Message) error {
		result := &schema.CheckResult{}
		if err := proto.Unmarshal(msg.Body, result); err != nil {
			log.WithError(err).Error("Error unmarshalling message from NSQ.")
			return err
		}

		logger := log.WithFields(log.Fields{
			"customer_id": result.CustomerId,
			"check_id":    result.CheckId,
			"bastion_id":  result.BastionId,
		})

		// TODO(greg): CheckResult objects should probably have a validator.
		if result.CustomerId == "" || result.CheckId == "" {
			logger.Error("Received invalid check result.")
			return nil
		}

		// TODO(greg): Once all bastions have been upgraded to include Bastion ID in
		// their check results, everything in this block can be deleted.
		// -----------------------------------------------------------------------
		if result.Version < 2 {
			// Set bastion ID
			bastionId, ok := bastionMap[result.CustomerId]
			if !ok {
				resp, err := kapi.Get(context.Background(), fmt.Sprintf("/opsee.co/routes/%s", result.CustomerId), nil)
				if err != nil {
					log.WithError(err).Error("Error getting bastion route from etcd.")
					return err
				}

				if len(resp.Node.Nodes) < 1 {
					log.Error("No bastion found for result in etcd.")
					// When we don't find a bastion for this customer, we just drop their results.
					// This isn't a problem after all customers are upgraded.
					return nil
				}

				bastionPath := resp.Node.Nodes[0].Key
				routeParts := strings.Split(bastionPath, "/")
				if len(routeParts) != 5 {
					log.WithError(err).Errorf("Unexpected route length: %d", len(routeParts))
					return err
				}
				bastionId = routeParts[4]
			}
			result.BastionId = bastionId
		}
		// -----------------------------------------------------------------------

		// For now, the region is just static, because we only have dynamodb in one region.

		task := worker.NewCheckWorker(db, result)
		_, err = task.Execute()
		if err != nil {
			logger.WithError(err).Error("Error executing task.")
			return err
		}

		go func(r *schema.CheckResult, logger *log.Entry) {
			err := s3Store.PutResult(r)
			if err != nil {
				logger.WithFields(log.Fields{"bucket_name": s3Store.BucketName}).WithError(err).Error("Error putting result to s3")
			}
		}(result, logger)

		go func(r *schema.CheckResult, logger *log.Entry) {
			err := dynamo.PutResult(r)
			if err != nil {
				logger.WithError(err).Error("Error putting result to dynamodb")
			}
		}(result, logger)

		checkResultsHandled.Inc()
		return nil
	})

	checks.AddHook(func(newStateID checks.StateId, state *checks.State, result *schema.CheckResult) {
		logger := log.WithFields(log.Fields{
			"customer_id":           state.CustomerId,
			"check_id":              state.CheckId,
			"min_failing_count":     state.MinFailingCount,
			"min_failing_time":      state.MinFailingTime,
			"failing_count":         state.FailingCount,
			"failing_time_s":        state.TimeInState().Seconds(),
			"old_state":             state.State,
			"new_state":             newStateID.String(),
			"result.response_count": len(result.Responses),
			"result.passing":        result.Passing,
			"result.failing_count":  result.FailingCount(),
			"result.timestamp":      result.Timestamp.String(),
		})

		checkStore := store.NewCheckStore(db)
		logEntry, err := checkStore.CreateStateTransitionLogEntry(state.CheckId, state.CustomerId, state.Id, newStateID)
		if err != nil {
			logger.WithError(err).Error("Error creating StateTransitionLogEntry")
		}

		logger.Infof("Created StateTransitionLogEntry: %d", logEntry.Id)
	})

	publishToNSQ := func(result *schema.CheckResult) {
		logger := log.WithFields(log.Fields{
			"customer_id": result.CustomerId,
			"check_id":    result.CheckId,
		})

		resultBytes, err := proto.Marshal(result)
		if err != nil {
			logger.WithError(err).Error("Unable to marshal CheckResult to protobuf")
		}
		if err := producer.Publish("alerts", resultBytes); err != nil {
			logger.WithError(err).Error("Error publishing alert to NSQ.")
		}
	}

	// TODO(greg): We should be able to set hooks on transitions from->to specific
	// states. Not have to guard in the transition function.
	//
	// transition functions need to be able to signal that we couldn't transition state.
	// in which case we should requeue the message. this could be due to a temporary SQS
	// failure or an error with the result. maybe logging/instrumenting this is enough?
	checks.AddStateHook(checks.StateOK, func(id checks.StateId, state *checks.State, result *schema.CheckResult) {
		logger := log.WithFields(log.Fields{
			"customer_id":       state.CustomerId,
			"check_id":          state.CheckId,
			"min_failing_count": state.MinFailingCount,
			"min_failing_time":  state.MinFailingTime,
			"failing_count":     state.FailingCount,
			"failing_time_s":    state.TimeInState().Seconds(),
			"old_state":         state.State,
			"new_state":         id.String(),
		})

		logger.Infof("check transitioned to passing")
		// We go FAIL -> PASS_WAIT -> OK or WARN
		if state.Id == checks.StatePassWait && id == checks.StateOK {
			publishToNSQ(result)
		}
	})

	checks.AddStateHook(checks.StateWarn, func(id checks.StateId, state *checks.State, result *schema.CheckResult) {
		logger := log.WithFields(log.Fields{
			"customer_id":       state.CustomerId,
			"check_id":          state.CheckId,
			"min_failing_count": state.MinFailingCount,
			"min_failing_time":  state.MinFailingTime,
			"failing_count":     state.FailingCount,
			"failing_time_s":    state.TimeInState().Seconds(),
			"old_state":         state.State,
			"new_state":         id.String(),
		})

		logger.Infof("check transitioned to warning")
		// We go FAIL -> PASS_WAIT -> OK or WARN
		if state.Id == checks.StatePassWait && id == checks.StateWarn {
			publishToNSQ(result)
		}
	})

	checks.AddStateHook(checks.StateFail, func(id checks.StateId, state *checks.State, result *schema.CheckResult) {
		logger := log.WithFields(log.Fields{
			"customer_id":       state.CustomerId,
			"check_id":          state.CheckId,
			"min_failing_count": state.MinFailingCount,
			"min_failing_time":  state.MinFailingTime,
			"failing_count":     state.FailingCount,
			"failing_time_s":    state.TimeInState().Seconds(),
			"old_state":         state.State,
			"new_state":         id.String(),
		})

		logger.Infof("check transitioned to fail")
		if state.Id == checks.StateFailWait && id == checks.StateFail {
			publishToNSQ(result)
		}
	})

	if err := consumer.Start(); err != nil {
		log.WithError(err).Fatal("Failed to start consumer.")
	}

	<-sigChan

	consumer.Stop()
}
