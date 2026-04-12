variable "environment" {
  description = "Deployment environment: dev, staging, or prod"
  type        = string
}

variable "docker_username" {
  description = "Docker Hub username used to pull images"
  type        = string
}

variable "image_tag" {
  description = "Docker image tag to deploy (e.g. prod, staging, dev, or a git SHA)"
  type        = string
  default     = "prod"
}

variable "db_password" {
  description = "PostgreSQL password (inject from CI/CD secret, never hardcode)"
  type        = string
  sensitive   = true
}

variable "db_user" {
  description = "PostgreSQL username"
  type        = string
  default     = "appuser"
}

variable "db_name" {
  description = "PostgreSQL database name"
  type        = string
  default     = "votes"
}

variable "vote_port" {
  description = "Host port to expose the vote service on"
  type        = number
  default     = 8080
}

variable "result_port" {
  description = "Host port to expose the result service on"
  type        = number
  default     = 4000
}
