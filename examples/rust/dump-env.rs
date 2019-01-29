// Standard CGI(ish) environment variables, as defined in
// http://tools.ietf.org/html/rfc3875

use std::env;

const NAMES: &'static [&'static str] = &[
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
  "HTTPS",
];

fn main() {
  for key in NAMES {
    let value = env::var(key).unwrap_or(String::from("<unset>"));
    println!("{}={}", key, value);
  }
  for (key, value) in env::vars() {
    if key.starts_with("HTTP_") {
      println!("{}={}", key, value);
    }
  }
}
