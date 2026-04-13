package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/cenkalti/backoff/v4"
	_ "github.com/lib/pq"

	kingpin "github.com/alecthomas/kingpin/v2"

	"github.com/IBM/sarama"
)

var (
	brokerList        = kingpin.Flag("brokerList", "List of brokers to connect").Default(getEnv("KAFKA_BROKERS", "kafka:9092")).Strings()
	topic             = kingpin.Flag("topic", "Topic name").Default("votes").String()
	messageCountStart = kingpin.Flag("messageCountStart", "Message counter start from:").Int()
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	db := openDatabase()
	defer db.Close()

	pingDatabase(db)

	createTableStmt := `CREATE TABLE IF NOT EXISTS votes (id VARCHAR(255) NOT NULL UNIQUE, vote VARCHAR(255) NOT NULL)`
	if _, err := db.Exec(createTableStmt); err != nil {
		log.Panic(err)
	}

	master := getKafkaMaster()
	defer master.Close()

	consumer, err := master.ConsumePartition(*topic, 0, sarama.OffsetOldest)
	if err != nil {
		log.Panic(err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	doneCh := make(chan struct{})
	go func() {
		for {
			select {
			case err := <-consumer.Errors():
				fmt.Println(err)
			case msg := <-consumer.Messages():
				*messageCountStart++
				fmt.Printf("Received message: user %s vote %s\n", string(msg.Key), string(msg.Value))

				insertDynStmt := `insert into "votes"("id", "vote") values($1, $2) on conflict(id) do update set vote = $2`
				writeOp := func() error {
					_, err := db.Exec(insertDynStmt, string(msg.Key), string(msg.Value))
					return err
				}
				bo := backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second), 3)
				if err := backoff.RetryNotify(writeOp, bo, func(err error, d time.Duration) {
					log.Printf("DB write failed, retrying in %s: %v", d, err)
				}); err != nil {
					log.Println("DB write error after retries:", err)
				}
			case <-signals:
				fmt.Println("Interrupt is detected")
				doneCh <- struct{}{}
			}
		}
	}()
	<-doneCh
	log.Println("Processed", *messageCountStart, "messages")
}

func openDatabase() *sql.DB {
	host := getEnv("DB_HOST", "postgresql")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "appuser")
	password := getEnv("DB_PASSWORD", "")
	dbname := getEnv("DB_NAME", "votes")

	psqlconn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var db *sql.DB
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 500 * time.Millisecond
	bo.MaxInterval = 30 * time.Second

	err := backoff.RetryNotify(
		func() error {
			var openErr error
			db, openErr = sql.Open("postgres", psqlconn)
			return openErr
		},
		bo,
		func(err error, d time.Duration) {
			log.Printf("DB open failed, retrying in %s: %v", d, err)
		},
	)
	if err != nil {
		log.Panic("Could not open database:", err)
	}
	return db
}

func pingDatabase(db *sql.DB) {
	fmt.Println("Waiting for postgresql...")
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 500 * time.Millisecond
	bo.MaxInterval = 30 * time.Second

	err := backoff.RetryNotify(
		db.Ping,
		bo,
		func(err error, d time.Duration) {
			log.Printf("DB ping failed, retrying in %s: %v", d, err)
		},
	)
	if err != nil {
		log.Panic("Could not ping database:", err)
	}
	fmt.Println("Postgresql connected!")
}

func getKafkaMaster() sarama.Consumer {
	kingpin.Parse()
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	brokers := *brokerList

	fmt.Println("Waiting for kafka...")
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 500 * time.Millisecond
	bo.MaxInterval = 30 * time.Second

	var master sarama.Consumer
	err := backoff.RetryNotify(
		func() error {
			var connErr error
			master, connErr = sarama.NewConsumer(brokers, config)
			return connErr
		},
		bo,
		func(err error, d time.Duration) {
			log.Printf("Kafka connection failed, retrying in %s: %v", d, err)
		},
	)
	if err != nil {
		log.Panic("Could not connect to Kafka:", err)
	}
	fmt.Println("Kafka connected!")
	return master
}
