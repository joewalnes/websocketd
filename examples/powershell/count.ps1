#!/usr/bin/env pwsh

for ($i = 1 ; $i -le 10; $i++) {
    Write-Host $i
    Start-Sleep -Milliseconds 500
}
