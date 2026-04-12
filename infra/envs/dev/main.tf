terraform {
  required_version = ">= 1.6"

  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0"
    }
  }

  # Remote state — replace with your actual backend (S3, GCS, Terraform Cloud, etc.)
  # backend "s3" {
  #   bucket = "my-tf-state"
  #   key    = "microservices-demo/dev/terraform.tfstate"
  #   region = "us-east-1"
  # }
}

provider "docker" {
  # Connects to local Docker daemon by default.
  # For remote hosts: host = "ssh://user@remote-host"
}

module "stack" {
  source = "../../modules/microservices-stack"

  environment     = "dev"
  docker_username = var.docker_username
  image_tag       = var.image_tag
  db_password     = var.db_password
  vote_port       = 8080
  result_port     = 4000
}

variable "docker_username" {
  type = string
}

variable "image_tag" {
  type    = string
  default = "dev"
}

variable "db_password" {
  type      = string
  sensitive = true
}

output "vote_url"   { value = module.stack.vote_service_url }
output "result_url" { value = module.stack.result_service_url }
