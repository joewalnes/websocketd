open System
open System.Threading

[<EntryPoint>]
let main argv = 
    [| 1..10 |] |> Array.iter (Console.WriteLine >> (fun _ -> Thread.Sleep(1000)))

    0 // return an integer exit code
