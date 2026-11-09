(() => {
 const root = document.getElementById('wsd-bridge');
 const examples = {
  Bash: {files:[{name:'count.sh',code:String.raw`#!/usr/bin/env bash

for ((count = 1; count <= 10; count++)); do
  echo "$count"
  sleep 0.5
done`}],launch:'count.sh'},
  Python: {files:[{name:'count.py',code:String.raw`#!/usr/bin/env python3
import time

for count in range(1, 11):
    print(count, flush=True)
    time.sleep(0.5)`}],launch:'count.py'},
  'Node.js': {files:[{name:'count.js',code:String.raw`#!/usr/bin/env node
let count = 0;

const timer = setInterval(() => {
  console.log(++count);
  if (count === 10) clearInterval(timer);
}, 500);`}],launch:'count.js'},
  Rust: {files:[{name:'count.rs',code:String.raw`use std::io::{self, Write};
use std::thread;
use std::time::Duration;

fn main() {
    for count in 1..=10 {
        println!("{}", count);
        io::stdout().flush().unwrap();
        thread::sleep(Duration::from_millis(500));
    }
}`}],setup:'rustc count.rs -o count',command:'$ websocketd --port=8080 ./count'},
  Ruby: {files:[{name:'count.rb',code:String.raw`#!/usr/bin/env ruby
STDOUT.sync = true

(1..10).each do |count|
  puts count
  sleep 0.5
end`}],launch:'count.rb'},
  PHP: {files:[{name:'count.php',code:String.raw`#!/usr/bin/env php
<?php
for ($count = 1; $count <= 10; $count++) {
    echo $count, PHP_EOL;
    flush();
    usleep(500000);
}`}],launch:'count.php'},
  Perl: {files:[{name:'count.pl',code:String.raw`#!/usr/bin/env perl
use Time::HiRes qw(sleep);
$| = 1; # Autoflush stdout.

for my $count (1 .. 10) {
    print "$count\n";
    sleep 0.5;
}`}],launch:'count.pl'},
  C: {files:[{name:'count.c',code:String.raw`#include <stdio.h>
#include <unistd.h>

int main(void) {
    setbuf(stdout, NULL);
    for (int count = 1; count <= 10; count++) {
        printf("%d\n", count);
        usleep(500000);
    }
    return 0;
}`}],setup:'cc count.c -o count',command:'$ websocketd --port=8080 ./count'},
  'C#': {files:[{name:'Counter.cs',code:String.raw`using System;
using System.Threading;

class Counter {
    static void Main() {
        for (int count = 1; count <= 10; count++) {
            Console.WriteLine(count);
            Thread.Sleep(500);
        }
    }
}`}],setup:'mcs Counter.cs -out:Counter.exe',command:'$ websocketd --port=8080 mono Counter.exe'},
  Swift: {files:[{name:'count.swift',code:String.raw`import Foundation

for count in 1...10 {
    print(count)
    fflush(nil)
    Thread.sleep(forTimeInterval: 0.5)
}`}],setup:'swiftc count.swift -o count',command:'$ websocketd --port=8080 ./count'},
  'Anything': {files:[{name:'Any command-line program',code:String.raw`Read inbound messages from stdin.
Write outbound messages to stdout.

Each line of stdout becomes a message.

Read HTTP and connection details from
environment variables.`}],command:'$ websocketd --port=8080 ./my-program'}
 };
 const program = root.querySelector('#w-program');
 const command = root.querySelector('#w-command');
 const filename = root.querySelector('#w-filename');
 const tabs = root.querySelector('.w-tabs');
 let selected = 'Python';
 function selectLanguage(language) {
  selected = language;
  const example = examples[language];
  tabs.querySelectorAll('[data-lang]').forEach(button => button.setAttribute('aria-pressed',String(button.dataset.lang === language)));
  program.replaceChildren();
  example.files.forEach(file => {
   if(example.files.length > 1) {
    const label = document.createElement('div'); label.className='w-file-label'; label.textContent=file.name;
    program.appendChild(label);
   }
   const pre = document.createElement('pre');pre.className='w-code';pre.textContent=file.code;program.appendChild(pre);
  });
  const commands = [example.setup && '$ '+example.setup, ...(example.command ? [example.command] : ['$ chmod +x '+example.launch, '$ websocketd --port=8080 ./'+example.launch])].filter(Boolean);
  command.textContent = commands.join('\n');
  filename.textContent = example.files.map(file=>file.name).join(' + ');
 }
 tabs.replaceChildren();
 Object.keys(examples).forEach(language=>{
  const button=document.createElement('button');button.type='button';button.className='cursor-interaction';button.dataset.lang=language;button.textContent=language==='Anything'?'In fact… anything!':language;
  button.addEventListener('click',()=>selectLanguage(language));tabs.appendChild(button);
 });
 const previous=root.querySelector('#w-tabs-prev'),next=root.querySelector('#w-tabs-next');
 function updateScroll(){const overflowing=tabs.scrollWidth>tabs.clientWidth+2;previous.hidden=!overflowing;next.hidden=!overflowing;previous.disabled=tabs.scrollLeft<2;next.disabled=tabs.scrollLeft+tabs.clientWidth>=tabs.scrollWidth-2;}
 function moveTabs(direction){tabs.scrollBy({left:direction*Math.max(140,tabs.clientWidth*.65),behavior:matchMedia('(prefers-reduced-motion: reduce)').matches?'instant':'smooth'});}
 previous.addEventListener('click',()=>moveTabs(-1));next.addEventListener('click',()=>moveTabs(1));tabs.addEventListener('scroll',updateScroll);new ResizeObserver(updateScroll).observe(tabs);
 selectLanguage(selected);updateScroll();
 root.querySelectorAll('a[href^="#w-"]').forEach(link=>link.addEventListener('click',event=>{event.preventDefault();root.querySelector(link.getAttribute('href')).scrollIntoView({behavior:matchMedia('(prefers-reduced-motion: reduce)').matches?'instant':'smooth',block:'start'});}));
})();
