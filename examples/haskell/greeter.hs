#!/usr/bin/env runhaskell

import Control.Monad (unless)
import System.IO (hFlush, stdout, stdin, hIsEOF)

-- | For each line FOO received on STDIN, respond with "Hello FOO!".
main :: IO ()
main = do
  eof <- hIsEOF stdin
  unless eof $ do
    line <- getLine
    putStrLn $ "Hello " ++ line ++ "!"
    hFlush stdout
    main
