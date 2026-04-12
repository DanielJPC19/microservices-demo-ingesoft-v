terraform {
  required_version = ">= 1.6"

  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0"
    }
  }
}

provider "docker" {}

module "stack" {
  source = "../../modules/microservices-stack"

  environment     = "prod"
  docker_username = var.docker_username
  image_tag       = var.image_tag
  db_password     = var.db_password
  vote_port       = 80
  result_port     = 3000
}

variable "docker_username" {
  type = string
}

# image_tag is set by CI/CD to a git SHA for versioned, rollback-capable deploys.
# To roll back: set this to a previous SHA and re-apply.
variable "image_tag" {
  type    = string
  default = "prod"
}

variable "db_password" {
  type      = string
  sensitive = true
}

output "vote_url"   { value = module.stack.vote_service_url }
output "result_url" { value = module.stack.result_service_url }
