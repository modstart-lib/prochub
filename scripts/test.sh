#!/bin/bash

echo "Starting test script..."

# echo arguments passed to the script
echo "Arguments: $@"

# echo current working directory
echo "PWD: $(pwd)"

# loop 5 times
for i in {1..5}; do
    sleep 2
    echo "Test iteration $i"
done

echo "Test script completed."