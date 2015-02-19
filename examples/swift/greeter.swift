#!/usr/bin/env xcrun -sdk macosx swift

import Foundation

while(true){
  var stdin = NSFileHandle.fileHandleWithStandardInput().availableData
  var line  = NSString(data: stdin, encoding: NSUTF8StringEncoding)!
  var name  = line.stringByTrimmingCharactersInSet(NSCharacterSet.newlineCharacterSet())
  println("Hello \(name)!")
  fflush(__stdoutp)
}
