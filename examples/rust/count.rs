use std::io::{self, Write};
use std::{thread, time};

// Simple example script that counts to 10 at ~2Hz, then stops.
fn main() {
  for i in 1..11 {
    println!("{}", i);
    io::stdout().flush().ok().expect("Could not flush stdout");
    thread::sleep(time::Duration::from_millis(500));
  }
}
