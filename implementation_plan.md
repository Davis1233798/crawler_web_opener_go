# Implementation Plan - IaC with Terraform

## Goal Description
Replace manual `gcloud` setup scripts with Terraform configuration to provision GCP infrastructure. This includes enabling APIs, creating the Artifact Registry repository, and provisioning the Mother VM.

## User Review Required
> [!IMPORTANT]
> - **Terraform Installed**: The user must have Terraform installed locally or use Cloud Shell.
> - **State Management**: By default, state will be local (`terraform.tfstate`). For production, a GCS backend is recommended but we will stick to local for simplicity.
> - **Service Account**: Terraform needs credentials (usually `gcloud auth application-default login`).

## Proposed Changes

### Branch: `iac-terraform` (derived from `no-proxy-docker-gcp`)

#### [NEW] [terraform/main.tf](file:///c:/Users/solidityDeveloper/crawler_web_opener/crawler-go/terraform/main.tf)
- Provider configuration (google).
- Enable APIs: `artifactregistry.googleapis.com`, `compute.googleapis.com`.
- Create Artifact Registry Repository: `crawler-repo`.

#### [NEW] [terraform/compute.tf](file:///c:/Users/solidityDeveloper/crawler_web_opener/crawler-go/terraform/compute.tf)
- Create Service Account for Mother VM.
- Create Compute Instance (Mother VM) with startup script to clone repo and install tools.

#### [NEW] [terraform/variables.tf](file:///c:/Users/solidityDeveloper/crawler_web_opener/crawler-go/terraform/variables.tf)
- Variables for `project_id`, `region`, `zone`.

#### [NEW] [terraform/outputs.tf](file:///c:/Users/solidityDeveloper/crawler_web_opener/crawler-go/terraform/outputs.tf)
- Output Mother VM IP and connection command.

#### [NEW] [terraform/README.md](file:///c:/Users/solidityDeveloper/crawler_web_opener/crawler-go/terraform/README.md)
- Instructions for `terraform init`, `plan`, `apply`.

## Verification Plan
- Run `terraform init` and `terraform validate` to check syntax.
- User to run `terraform apply` to provision resources.
