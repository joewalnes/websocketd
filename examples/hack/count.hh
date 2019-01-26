#!/usr/bin/hhvm
<?hh // strict

use function HH\Lib\Experimental\IO\request_output;
use function usleep;

// Simple example script that counts to 10 at ~2Hz, then stops.

<<__EntryPoint>>
async function count(): Awaitable<void> {
  $output = request_output();
  for ($count = 1; $count <= 10; $count++) {
    await $output->writeAsync(
      Str\format("%d\n",$count)
    );
 
    // usleep is builtin, it is not an async builtin - so it also must block the main request thread
    usleep(500000);
  }
  
  // flush output
  await $output->flushAsync();
}
