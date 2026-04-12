output "vote_service_url" {
  description = "URL for the vote service"
  value       = "http://localhost:${var.vote_port}"
}

output "result_service_url" {
  description = "URL for the result service"
  value       = "http://localhost:${var.result_port}"
}

output "network_name" {
  description = "Docker network used by all services"
  value       = docker_network.app_network.name
}
