#!/usr/bin/env pwsh

# Standard CGI(ish) environment variables, as defined in
# http://tools.ietf.org/html/rfc3875
$varNames = @(
    "AUTH_TYPE",
    "CONTENT_LENGTH",
    "CONTENT_TYPE",
    "GATEWAY_INTERFACE",
    "PATH_INFO",
    "PATH_TRANSLATED",
    "QUERY_STRING",
    "REMOTE_ADDR",
    "REMOTE_HOST",
    "REMOTE_IDENT",
    "REMOTE_PORT",
    "REMOTE_USER",
    "REQUEST_METHOD",
    "REQUEST_URI",
    "SCRIPT_NAME",
    "SERVER_NAME",
    "SERVER_PORT",
    "SERVER_PROTOCOL",
    "SERVER_SOFTWARE",
    "UNIQUE_ID",
    "HTTPS"
)

$environmentVariables = [Environment]::GetEnvironmentVariables()

foreach ($key in $varNames) {
    $value = $environmentVariables.Item($key)
    if ([string]::IsNullOrEmpty($value)) {
        $value = "<unset>"
    }
    Write-Host "$key=$value"
}

# Additional HTTP headers
foreach ($item in $environmentVariables.GetEnumerator()) {
    $key, $value = $item.Key, $item.Value
    if ($key.StartsWith("HTTP_")) {
        Write-Host "$key=$value"
    }
}
