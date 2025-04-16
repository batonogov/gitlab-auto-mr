#!/bin/bash

set -e

# Install Go version of gitlab-auto-mr
install_local() {
  echo "Building and installing from source..."
  go mod tidy
  go build -o gitlab-auto-mr
  chmod +x gitlab-auto-mr
  sudo mv gitlab-auto-mr /usr/local/bin/
  echo "Installed gitlab-auto-mr to /usr/local/bin/gitlab-auto-mr"
}

# Main script
echo "GitLab Auto MR - Go Version"
install_local

echo "Done! You can now use gitlab-auto-mr command"