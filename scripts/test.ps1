#!/usr/bin/env pwsh

Write-Host "Starting test script..."

# echo arguments passed to the script
Write-Host "Arguments: $args"

# echo current working directory
Write-Host "PWD: $(Get-Location)"

# loop 5 times
for ($i = 1; $i -le 5; $i++) {
    Start-Sleep -Seconds 2
    Write-Host "Test iteration $i"
}

Write-Host "Test script completed."
