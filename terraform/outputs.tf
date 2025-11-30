output "mother_vm_name" {
  value = google_compute_instance.mother_vm.name
}

output "mother_vm_external_ip" {
  value = google_compute_instance.mother_vm.network_interface[0].access_config[0].nat_ip
}

output "ssh_command" {
  value = "gcloud compute ssh ${google_compute_instance.mother_vm.name} --zone=${var.zone}"
}

output "artifact_registry_url" {
  value = "${var.region}-docker.pkg.dev/${var.project_id}/${var.repo_name}"
}
