use std::io::{self, Write};

// For each line FOO received on STDIN, respond with "Hello FOO!".
fn main() {
  loop {
    let mut msg = String::new();
    io::stdin()
      .read_line(&mut msg)
      .expect("Failed to read line");
    let msg = msg.trim();
    println!("Hello {}!", msg);
    io::stdout().flush().ok().expect("Could not flush stdout");
  }
}
