#!/usr/bin/env runhaskell

import Control.Monad (forM_)
import Control.Concurrent (threadDelay)

-- | Count from 1 to 10 with a sleep
main :: IO ()
main = forM_ [1 :: Int .. 10] $ \count -> do
  print count
  threadDelay 500000
