#!/usr/bin/env pwsh

# For each line FOO received on STDIN, respond with "Hello FOO!".
while ($true) {
    $line = Read-Host
    Write-Host "Hello $line!"
}
