#!/usr/bin/env xcrun -sdk macosx swift

import AppKit

for index in 1...10 {
  println(index)
  
  // Flush output
  fflush(__stdoutp)
  
  NSThread.sleepForTimeInterval(0.5)
}