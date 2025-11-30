param (
    [Parameter(Mandatory = $true)]
    [string]$ProjectId
)

$ImageName = "us-central1-docker.pkg.dev/$ProjectId/crawler-repo/crawler-no-proxy:latest"

# 1. Build and Push Docker Image
Write-Host "Building and pushing Docker image to $ImageName..."
Push-Location ..
docker build -t $ImageName .
if ($LASTEXITCODE -ne 0) {
    Write-Error "Docker build failed"
    Pop-Location
    exit 1
}

# Configure docker credential helper for gcloud
gcloud auth configure-docker us-central1-docker.pkg.dev --quiet

docker push $ImageName
if ($LASTEXITCODE -ne 0) {
    Write-Error "Docker push failed"
    Pop-Location
    exit 1
}
Pop-Location

# 2. Define Regions
$Regions = @("us-central1", "europe-west1", "asia-northeast1")

Write-Host "Deploying to regions: $($Regions -join ', ')"

# 3. Deploy to each region
foreach ($Region in $Regions) {
    $Timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $InstanceName = "crawler-$Region-$Timestamp"
    $Zone = "$Region-a"

    Write-Host "Deploying instance $InstanceName in $Zone..."

    # Startup script content
    $StartupScript = @"
#!/bin/bash
gcloud auth configure-docker us-central1-docker.pkg.dev --quiet
docker run --rm -e THREADS=2 -e DURATION=60 -e HEADLESS=true $ImageName
echo 'Crawler finished. Self-destructing in 60s...'
sleep 60
gcloud compute instances delete $InstanceName --zone=$Zone --quiet
"@

    gcloud compute instances create $InstanceName `
        --project=$ProjectId `
        --zone=$Zone `
        --image-family=cos-stable `
        --image-project=cos-cloud `
        --machine-type=e2-micro `
        --scopes=https://www.googleapis.com/auth/cloud-platform `
        --metadata=startup-script="$StartupScript" `
        --tags=crawler `
        --preemptible

    if ($LASTEXITCODE -eq 0) {
        Write-Host "Successfully deployed to $Region" -ForegroundColor Green
    }
    else {
        Write-Host "Failed to deploy to $Region" -ForegroundColor Red
    }
}

Write-Host "Deployment complete. Instances will self-destruct after running."
