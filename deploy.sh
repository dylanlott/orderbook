#!/bin/bash

# Set the remote server details
REMOTE_HOST="www.dylanlott.com"
REMOTE_USER="root"
REMOTE_DIR="/home/shakezula/orderbook"

# Set the Docker container details
DOCKER_IMAGE="orderbook"
DOCKER_CONTAINER_NAME="orderbook-server"

# Sync the current directory to the remote server
rsync -avz --delete ./ "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR}"

# SSH into the remote server and build the Docker container
ssh "${REMOTE_USER}@${REMOTE_HOST}" << EOF
  # Change to the copied directory
  cd "${REMOTE_DIR}"

  # Build the Docker image
  docker build -t "${DOCKER_IMAGE}" .

  # Remove existing container if it exists
  docker rm -f "${DOCKER_CONTAINER_NAME}" || true

  # Run the Docker container
  docker run -d --name "${DOCKER_CONTAINER_NAME}" -p 1323:1323 "${DOCKER_IMAGE}"
EOF
