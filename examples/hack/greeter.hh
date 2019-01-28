#!/usr/bin/hhvm
<?hh // strict

use namespace HH\Lib\Str;
use namespace HH\Lib\Experimental\IO;

<<__EntryPoint>>
async function greeter(): Awaitable<noreturn> {
  // For each line FOO received on STDIN, respond with "Hello FOO!".
  $input = IO\request_input();
  $output = IO\request_output();
  while(!$input->isEndOfFile()) {
    await $ouput->writeAsync(
      Str\format("Hello %s!\n", await $input->readLineAsync())
    );
  }
    
  // flush output
  await $output->flushAsync();
  
  exit(0);
}
