#!/usr/bin/env -S qjs --module
import * as std from "std";

let line;
while ((line = std.in.getline()) != null) {
  console.log("RCVD: " + line)
  std.out.flush();
}
