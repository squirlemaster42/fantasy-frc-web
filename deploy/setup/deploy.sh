#!/bin/bash
set -e

echo "Starting Fantasy FRC Deploy"
cd /home/user/fantasy-frc-web/server

echo "Pulling latest code..."
git pull

echo "Building..."
make build

echo "Stopping service..."
sudo systemctl stop fantasyfrc

echo "Deploying binary..."
sudo cp /home/user/fantasy-frc-web/server/server /usr/local/bin/fantasyfrcserver

echo "Starting service..."
sudo systemctl start fantasyfrc

echo "Deploy complete"
