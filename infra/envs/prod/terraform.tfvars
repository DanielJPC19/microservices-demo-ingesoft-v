# Production environment variables
# image_tag is overridden by CI/CD on every deploy (set to the git SHA).
# To roll back: update image_tag to a previous git SHA and commit to main.
# Sensitive values (db_password, docker_username) are injected by CI/CD as TF_VAR_* env vars.

image_tag = "prod"
