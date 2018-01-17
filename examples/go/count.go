package main

import (
  "fmt"
  "time"
  //"bufio"
  //"strings"
  //"os"
)

func main() {
  var message string
  for i := 0; i < 10; i++ {
    fmt.Println(i);
    time.Sleep(time.Second)
  }

  for {
    if n, _ := fmt.Scanln(&message); n > 0 {
      if message == "close" {
        break
      }
      fmt.Println(message)
    }
  }

  //inputReader := bufio.NewReader(os.Stdin)
  //
  //for message, _ = inputReader.ReadString('\n'); len(message)-1>0 ; message, _ = inputReader.ReadString('\n'){
  //  if message == "close" {
  //    os.Exit(2)
  //    break
  //  }
  //  fmt.Println(strings.Trim(message,"\n"))
  //}
}
