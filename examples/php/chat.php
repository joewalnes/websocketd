#!/usr/bin/php
<?php

$stdin = fopen('php://stdin', 'r');
$filename = __DIR__.'/chat.log';
echo 'Please enter your name:'.PHP_EOL;
$user = trim(fgets($stdin));
$message = '['.date('Y-m-d H:i:s').'] '.$user.' joined the chat'.PHP_EOL;
file_put_contents($filename, $message, FILE_APPEND);

echo '['.date('Y-m-d H:i:s').'] Welcome to the chat '.$user.'!'.PHP_EOL;
$pid = pcntl_fork();

if ($pid  == - 1 ) {
	die( 'could not fork' );
} else if ($pid) {
	while ($msg = fgets($stdin)) {
		$message = '['.date('Y-m-d H:i:s').'] '.$user.' '.$msg.PHP_EOL;
		file_put_contents($filename, $message, FILE_APPEND);
	}
	pcntl_wait($status);
} else {
	$lastmtime = null;
	$ftell = null;

	while (1) {
		$fp = fopen($filename,  'r');
		if ($fp) {
			$fstat = fstat($fp);
			$mtime = $fstat['mtime'];
			if (!$lastmtime) {
				fseek($fp, 0, SEEK_END);
				$lastmtime = $mtime;
				$ftell = ftell($fp);
			} else if ($lastmtime < $mtime) {
				$lastmtime = $mtime;
				fseek($fp, $ftell);
				while (!feof($fp) && ($line  =  fgets($fp, 4096)) !==  false ) {
					echo $line;
				}
				$ftell = ftell($fp);
			}
			fclose($fp);
		}
		sleep(1);
	}
}