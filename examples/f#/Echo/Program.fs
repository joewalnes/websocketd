open System

[<EntryPoint>]
let main argv = 
    let rec recLoop () =
        Console.ReadLine() |> Console.WriteLine
        recLoop()

    recLoop()
    
    0 // return an integer exit code
