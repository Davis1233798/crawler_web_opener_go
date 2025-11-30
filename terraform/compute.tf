# Service Account for Mother VM
resource "google_service_account" "mother_vm_sa" {
  account_id   = "crawler-mother-vm-sa"
  display_name = "Service Account for Crawler Mother VM"
}

# Grant permissions to SA (Editor role for simplicity in this demo, restrict in prod!)
resource "google_project_iam_member" "mother_vm_sa_editor" {
  project = var.project_id
  role    = "roles/editor"
  member  = "serviceAccount:${google_service_account.mother_vm_sa.email}"
}

# Mother VM Instance
resource "google_compute_instance" "mother_vm" {
  name         = "crawler-mother-vm"
  machine_type = "e2-micro"
  zone         = var.zone
  tags         = ["controller"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-11"
    }
  }

  network_interface {
    network = "default"
    access_config {
      # Ephemeral public IP
    }
  }

  service_account {
    email  = google_service_account.mother_vm_sa.email
    scopes = ["cloud-platform"]
  }

  metadata_startup_script = templatefile("${path.module}/mother_startup.sh", {
    project_id = var.project_id
  })

  depends_on = [google_project_service.compute]
}
