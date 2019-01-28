#!/usr/bin/hhvm
<?hh // strict

use namespace HH\Lib\Str;
use function HH\Lib\Experimental\IO\request_output;

// Simple example script that counts to 10 at ~2Hz, then stops.

<<__EntryPoint>>
async function count_to_ten(): Awaitable<noreturn> {
  $output = request_output();
  for ($count = 1; $count <= 10; $count++) {
    await $output->writeAsync(
      Str\format("%d\n",$count)
    );

    HH\Asio\usleep(500000);
  }

  // flush output
  await $output->flushAsync();

  exit(0);
}
