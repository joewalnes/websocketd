#!/usr/bin/perl

#
# /usr/local/websocketd/ws_perl.pl
#
# Once started up, this script keeps running and checking for the existance of files.  If a file exists, it 
#   will send it to browser.
#

use strict;
use Data::Dumper;
use POSIX;

# Autoflush output
use IO::Handle;
STDOUT->autoflush(1);

my $cnt = 0;
#
# Stay open
#
while (1) {
  $cnt++;

  #
  # Grab QUERY_STRING and parse for PHONE
  #  Assumes a single parameter/value pair
  #
  my $QS     = $ENV{QUERY_STRING};
  my @params = split('=', $QS);
  my $phone  = $params[1];

  # Filename is phone.txt
  my $filename  = '/tmp/messages/' . $phone  . '.txt';
  if ( -f $filename ) {
    open( my $readfile, "<", $filename);

    # read and print each line from file
    my $line;
    while ( $line = <$readfile> ){
      print "$line<br>";
    }
    close($readfile);

    # Erase file once printed
    unlink($filename);
  }

  # Sleep for 20 seconds before checking again 
  sleep 20;
}

1;
