terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0"
    }
  }
}

# ─────────────────────────────────────────────
# Shared network — all containers communicate
# over this network by service name (DNS).
# ─────────────────────────────────────────────
resource "docker_network" "app_network" {
  name = "votes-${var.environment}"
}

# ─────────────────────────────────────────────
# PostgreSQL
# ─────────────────────────────────────────────
resource "docker_image" "postgresql" {
  name         = "postgres:16"
  keep_locally = true
}

resource "docker_container" "postgresql" {
  name  = "postgresql-${var.environment}"
  image = docker_image.postgresql.image_id

  networks_advanced {
    name    = docker_network.app_network.name
    aliases = ["postgresql"]
  }

  env = [
    "POSTGRES_USER=${var.db_user}",
    "POSTGRES_PASSWORD=${var.db_password}",
    "POSTGRES_DB=${var.db_name}",
  ]

  healthcheck {
    test     = ["CMD-SHELL", "pg_isready -U ${var.db_user}"]
    interval = "10s"
    timeout  = "5s"
    retries  = 5
  }

  restart = "unless-stopped"
}

# ─────────────────────────────────────────────
# Apache Kafka (KRaft mode — no ZooKeeper)
# ─────────────────────────────────────────────
resource "docker_image" "kafka" {
  name         = "apache/kafka:3.7.0"
  keep_locally = true
}

resource "docker_container" "kafka" {
  name  = "kafka-${var.environment}"
  image = docker_image.kafka.image_id

  networks_advanced {
    name    = docker_network.app_network.name
    aliases = ["kafka"]
  }

  env = [
    "KAFKA_NODE_ID=1",
    "KAFKA_PROCESS_ROLES=broker,controller",
    "KAFKA_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093",
    "KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092",
    "KAFKA_CONTROLLER_QUORUM_VOTERS=1@localhost:9093",
    "KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER",
    "KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT",
    "KAFKA_AUTO_CREATE_TOPICS_ENABLE=true",
  ]

  restart = "unless-stopped"
}

# ─────────────────────────────────────────────
# Vote Service (Java / Spring Boot)
# ─────────────────────────────────────────────
resource "docker_image" "vote" {
  name         = "${var.docker_username}/vote:${var.image_tag}"
  keep_locally = false
}

resource "docker_container" "vote" {
  name  = "vote-${var.environment}"
  image = docker_image.vote.image_id

  networks_advanced {
    name    = docker_network.app_network.name
    aliases = ["vote"]
  }

  ports {
    internal = 8080
    external = var.vote_port
  }

  env = [
    "KAFKA_BOOTSTRAP_SERVERS=kafka:9092",
  ]

  depends_on = [docker_container.kafka]
  restart    = "unless-stopped"
}

# ─────────────────────────────────────────────
# Worker Service (Go)
# ─────────────────────────────────────────────
resource "docker_image" "worker" {
  name         = "${var.docker_username}/worker:${var.image_tag}"
  keep_locally = false
}

resource "docker_container" "worker" {
  name  = "worker-${var.environment}"
  image = docker_image.worker.image_id

  networks_advanced {
    name = docker_network.app_network.name
  }

  env = [
    "DB_HOST=postgresql",
    "DB_PORT=5432",
    "DB_USER=${var.db_user}",
    "DB_PASSWORD=${var.db_password}",
    "DB_NAME=${var.db_name}",
    "KAFKA_BROKERS=kafka:9092",
  ]

  depends_on = [docker_container.postgresql, docker_container.kafka]
  restart    = "unless-stopped"
}

# ─────────────────────────────────────────────
# Result Service (Node.js)
# ─────────────────────────────────────────────
resource "docker_image" "result" {
  name         = "${var.docker_username}/result:${var.image_tag}"
  keep_locally = false
}

resource "docker_container" "result" {
  name  = "result-${var.environment}"
  image = docker_image.result.image_id

  networks_advanced {
    name    = docker_network.app_network.name
    aliases = ["result"]
  }

  ports {
    internal = 80
    external = var.result_port
  }

  env = [
    "DB_HOST=postgresql",
    "DB_PORT=5432",
    "DB_USER=${var.db_user}",
    "DB_PASSWORD=${var.db_password}",
    "DB_NAME=${var.db_name}",
  ]

  depends_on = [docker_container.postgresql]
  restart    = "unless-stopped"
}
