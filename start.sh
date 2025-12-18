#!/bin/sh

# Start the platform in the background
echo "Starting platform..."
./platform > platform.log 2>&1 &

# Wait for platform to be ready
echo "Waiting for platform to initialize..."
sleep 5

# Start the bot
echo "Starting bot agent..."
./bot
