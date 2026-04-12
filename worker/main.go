package main

import (
	"log"
	"os"
	"os/signal"

	"database/sql"
	"fmt"

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
				if _, err := db.Exec(insertDynStmt, string(msg.Key), string(msg.Value)); err != nil {
					log.Println("DB write error:", err)
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
	for {
		db, err := sql.Open("postgres", psqlconn)
		if err == nil {
			return db
		}
	}
}

func pingDatabase(db *sql.DB) {
	fmt.Println("Waiting for postgresql...")
	for {
		if err := db.Ping(); err == nil {
			fmt.Println("Postgresql connected!")
			return
		}
	}
}

func getKafkaMaster() sarama.Consumer {
	kingpin.Parse()
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	brokers := *brokerList
	fmt.Println("Waiting for kafka...")
	for {
		master, err := sarama.NewConsumer(brokers, config)
		if err == nil {
			fmt.Println("Kafka connected!")
			return master
		}
	}
}
