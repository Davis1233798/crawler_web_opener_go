package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

func main() {
	image := flag.String("image", "", "Docker image to run")
	count := flag.Int("count", 1, "Number of VMs to create")
	duration := flag.Int("duration", 60, "Duration to run in seconds")
	project := flag.String("project", "", "GCP Project ID")
	zone := flag.String("zone", "us-central1-a", "GCP Zone")
	dryRun := flag.Bool("dry-run", false, "Print commands without executing")
	runOnce := flag.Bool("run-once", false, "Run tasks once and self-destruct")
	continuous := flag.Bool("continuous", false, "Run in a continuous loop (unattended mode)")
	interval := flag.Int("interval", 300, "Interval in seconds between batches in continuous mode")

	flag.Parse()

	if *image == "" || *project == "" {
		fmt.Println("Usage: gcp_runner -image <image> -project <project> [options]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	for {
		runBatch(*image, *project, *zone, *count, *duration, *dryRun, *runOnce)

		if !*continuous {
			break
		}
		log.Printf("Continuous mode: Waiting %d seconds before next batch...", *interval)
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

func runBatch(image, project, zone string, count, duration int, dryRun, runOnce bool) {
	var wg sync.WaitGroup
	instanceNames := make([]string, 0, count)

	// Create VMs
	log.Printf("Creating %d VMs...", count)
	for i := 0; i < count; i++ {
		instanceName := fmt.Sprintf("crawler-worker-%d-%d", time.Now().Unix(), i)
		instanceNames = append(instanceNames, instanceName)
		wg.Add(1)

		go func(name string) {
			defer wg.Done()
			createVM(name, image, project, zone, dryRun, runOnce)
		}(instanceName)
	}
	wg.Wait()

	if runOnce {
		log.Println("VMs created in Run-Once mode. They will self-destruct upon completion.")
		return
	}

	log.Printf("All VMs created. Running for %d seconds...", duration)
	time.Sleep(time.Duration(duration) * time.Second)

	// Delete VMs
	log.Println("Time's up! Deleting VMs...")
	for _, name := range instanceNames {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			deleteVM(n, project, zone, dryRun)
		}(name)
	}
	wg.Wait()
	log.Println("All VMs deleted. Batch done.")
}

func createVM(name, image, project, zone string, dryRun, runOnce bool) {
	// gcloud compute instances create-with-container <name> \
	// --project=<project> --zone=<zone> \
	// --container-image=<image> \
	// --container-env=NO_PROXY_MODE=true

	envVars := "NO_PROXY_MODE=true"
	if runOnce {
		envVars += ",RUN_ONCE=true,SELF_DESTRUCT=true"
	}

	args := []string{
		"compute", "instances", "create-with-container", name,
		"--project", project,
		"--zone", zone,
		"--container-image", image,
		"--container-env", envVars,
		"--quiet",                                                    // Non-interactive
		"--scopes", "https://www.googleapis.com/auth/cloud-platform", // Needed for self-destruct
	}

	cmd := exec.Command("gcloud", args...)
	if dryRun {
		log.Printf("[DRY RUN] %s", strings.Join(cmd.Args, " "))
		return
	}

	log.Printf("Creating VM %s...", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error creating VM %s: %v\nOutput: %s", name, err, string(output))
	} else {
		log.Printf("VM %s created successfully.", name)
	}
}

func deleteVM(name, project, zone string, dryRun bool) {
	args := []string{
		"compute", "instances", "delete", name,
		"--project", project,
		"--zone", zone,
		"--quiet",
	}

	cmd := exec.Command("gcloud", args...)
	if dryRun {
		log.Printf("[DRY RUN] %s", strings.Join(cmd.Args, " "))
		return
	}

	log.Printf("Deleting VM %s...", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error deleting VM %s: %v\nOutput: %s", name, err, string(output))
	} else {
		log.Printf("VM %s deleted successfully.", name)
	}
}
