{-# OPTIONS_GHC -Wall -Werror #-}
module TextMate where

import Data.List (intercalate)

-- structure of the textmate json

type ScopeName = String
type Regex = String
type CaptureKey = String
type Capture = (CaptureKey, ScopeName)
type RepoKey = String
type Named = (RepoKey, TmPattern)

data TmPattern
  = TmMatch   (Maybe ScopeName) Regex [Capture]
  | TmRegion  (Maybe ScopeName) Regex Regex [Capture] [Capture] [TmPattern]
  --           name              begin end   beginCaps  endCaps   inner
  | TmInclude RepoKey

-- Serialization from objects to raw JSON String

jsonStr :: String -> String
jsonStr s = "\"" ++ concatMap esc s ++ "\""
  where
    esc '"'  = "\\\""
    esc '\\' = "\\\\"
    esc '\n' = "\\n"
    esc c    = [c]

ind :: Int -> String
ind n = replicate (n * 2) ' '

capJson :: Int -> Capture -> String
capJson d (k, v) = ind d ++ jsonStr k ++ ": { " ++ jsonStr "name" ++ ": " ++ jsonStr v ++ " }"

patJson :: Int -> TmPattern -> String
patJson d (TmInclude ref) =
  ind d ++ "{ " ++ jsonStr "include" ++ ": " ++ jsonStr ("#" ++ ref) ++ " }"
patJson d (TmMatch name match caps) =
  ind d ++ "{\n"
  ++ maybe "" (\n -> ind (d+1) ++ jsonStr "name" ++ ": " ++ jsonStr n ++ ",\n") name
  ++ ind (d+1) ++ jsonStr "match" ++ ": " ++ jsonStr match
  ++ (if null caps then "\n"
      else ",\n" ++ ind (d+1) ++ jsonStr "captures" ++ ": {\n"
           ++ intercalate ",\n" (map (capJson (d+2)) caps) ++ "\n"
           ++ ind (d+1) ++ "}\n")
  ++ ind d ++ "}"
patJson d (TmRegion name begin end bCaps eCaps pats) =
  ind d ++ "{\n"
  ++ maybe "" (\n -> ind (d+1) ++ jsonStr "name" ++ ": " ++ jsonStr n ++ ",\n") name
  ++ ind (d+1) ++ jsonStr "begin" ++ ": " ++ jsonStr begin ++ ",\n"
  ++ ind (d+1) ++ jsonStr "end" ++ ": " ++ jsonStr end ++ ",\n"
  ++ ind (d+1) ++ jsonStr "beginCaptures" ++ ": {\n"
  ++ intercalate ",\n" (map (capJson (d+2)) bCaps) ++ "\n"
  ++ ind (d+1) ++ "},\n"
  ++ ind (d+1) ++ jsonStr "endCaptures" ++ ": {\n"
  ++ intercalate ",\n" (map (capJson (d+2)) eCaps) ++ "\n"
  ++ ind (d+1) ++ "},\n"
  ++ ind (d+1) ++ jsonStr "patterns" ++ ": [\n"
  ++ intercalate ",\n" (map (patJson (d+2)) pats) ++ "\n"
  ++ ind (d+1) ++ "]\n"
  ++ ind d ++ "}"

repoJson :: Int -> [Named] -> String
repoJson d entries =
  ind d ++ "{\n"
  ++ intercalate ",\n" (map entry entries) ++ "\n"
  ++ ind d ++ "}"
  where
    entry (k, p) = ind (d+1) ++ jsonStr k ++ ": "
                   ++ drop (length (ind (d+1))) (patJson (d+1) p)

grammarJson :: [Named] -> String
grammarJson entries =
  "{\n"
  ++ ind 1 ++ jsonStr "$schema" ++ ": " ++ jsonStr "https://raw.githubusercontent.com/martinring/tmlanguage/master/tmlanguage.json" ++ ",\n"
  ++ ind 1 ++ jsonStr "scopeName" ++ ": " ++ jsonStr "source.gotmpl" ++ ",\n"
  ++ ind 1 ++ jsonStr "fileTypes" ++ ": [" ++ jsonStr "tmpl" ++ ", " ++ jsonStr "gotmpl" ++ "],\n"
  ++ ind 1 ++ jsonStr "patterns" ++ ": [\n"
  ++ patJson 2 (TmInclude "comment") ++ ",\n"
  ++ patJson 2 (TmInclude "action") ++ "\n"
  ++ ind 1 ++ "],\n"
  ++ ind 1 ++ jsonStr "repository" ++ ": " ++ repoJson 1 entries ++ "\n"
  ++ "}\n"
