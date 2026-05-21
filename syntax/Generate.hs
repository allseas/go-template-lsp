{-# OPTIONS_GHC -Wall -Werror #-} -- ensures all cases are covered at compilation
module Main where

import Data.List (intercalate)
import System.Directory (createDirectoryIfMissing)

import Grammar
    ( VariableOp(..),
      Term(..),
      LoopAction(..),
      ActionBody(..),
      TemplateNode(..),
      keywords,
      builtinFunctions )
import TextMate ( Named, TmPattern(..), grammarJson )

-- Regex helpers

escRegex :: String -> String
escRegex = concatMap (\c -> if c `elem` ("{}()[]\\.*+?^$|" :: String) then ['\\',c] else [c])

wordAlt :: [String] -> String
wordAlt ws = "\\b(" ++ intercalate "|" ws ++ ")\\b"

dOpen :: String
dOpen = escRegex "{{" ++ "\\-?"

dClose :: String
dClose = "\\-?" ++ escRegex "}}"

-- regex specifications for all grammar elements
-- scopes are taken from 2026-dark.json VSCode theme

templateNodePatterns :: TemplateNode -> [Named]
templateNodePatterns TextNode = []
templateNodePatterns CommentNode =
  [("comment", TmRegion (Just "comment.block.gotmpl")
      (dOpen ++ "\\s*/\\*")
      ("\\*/\\s*" ++ dClose)
      [("0", "punctuation.definition.comment.begin.gotmpl")]
      [("0", "punctuation.definition.comment.end.gotmpl")]
      [])]
templateNodePatterns ActionNode =
  [("action", TmRegion (Just "meta.embedded.gotmpl")
      dOpen dClose
      [("0", "punctuation.definition.embedded.begin.gotmpl")]
      [("0", "punctuation.definition.embedded.end.gotmpl")]
      innerIncludes)]

actionBodyPatterns :: ActionBody -> [Named]
actionBodyPatterns PipelineAction =
  [("pipe", TmMatch (Just "keyword.operator.pipe.gotmpl") "\\|" [])]
actionBodyPatterns IfAction = []
actionBodyPatterns RangeAction =
  [("comma", TmMatch (Just "punctuation.separator.comma.gotmpl") "," [])]
actionBodyPatterns WithAction = []
actionBodyPatterns TemplateInvokeAction =
  [("template-name", TmMatch Nothing "(?<=template|block|define)\\s+(\"[^\"]*\")"
      [("1", "entity.name.function.gotmpl")])]
actionBodyPatterns BlockAction = []  -- name covered by template-name lookbehind
actionBodyPatterns DefineAction = [] -- name covered by template-name lookbehind

loopActionPatterns :: LoopAction -> [Named]
loopActionPatterns BreakAction = []    -- covered by keywords list
loopActionPatterns ContinueAction = [] -- covered by keywords list


termPatterns :: Term -> [Named]
termPatterns BoolLiteral =
  [("boolean", TmMatch (Just "constant.language.boolean.gotmpl") "\\b(true|false)\\b" [])]
termPatterns NumberLiteral =
  let r = "\\b(?:0[xX][0-9a-fA-F_]+|0[oO][0-7_]+|0[bB][01_]+|[0-9][0-9_]*(?:\\.[0-9_]+)?(?:[eE][+-]?[0-9_]+)?)i?\\b"
  in [("number", TmMatch (Just "constant.numeric.gotmpl") r [])]
termPatterns StringLiteral =
  [("string-double", TmRegion (Just "string.quoted.double.gotmpl")
      "\"" "\""
      [("0", "punctuation.definition.string.begin.gotmpl")]
      [("0", "punctuation.definition.string.end.gotmpl")]
      [TmMatch (Just "constant.character.escape.gotmpl") "\\\\." []])
  ,("string-raw", TmRegion (Just "string.quoted.other.raw.gotmpl")
      "`" "`"
      [("0", "punctuation.definition.string.begin.gotmpl")]
      [("0", "punctuation.definition.string.end.gotmpl")]
      [])]
termPatterns CharLiteral =
  [("char", TmMatch (Just "string.quoted.single.gotmpl") "'(?:\\\\.|[^'])'" [])]
termPatterns NilLiteral =
  [("nil", TmMatch (Just "constant.language.nil.gotmpl") "\\bnil\\b" [])]
termPatterns DotLiteral =
  [("dot", TmMatch (Just "variable.language.dot.gotmpl") "(?<![a-zA-Z_\\w])\\.(?![a-zA-Z_\\w])" [])]
termPatterns FieldNode =
  [("field", TmMatch (Just "variable.other.member.gotmpl") "\\.[a-zA-Z_]\\w*" [])]
termPatterns ChainNode =
  [("parentheses", TmRegion Nothing
      "\\(" "\\)"
      [("0", "punctuation.section.parens.begin.gotmpl")]
      [("0", "punctuation.section.parens.end.gotmpl")]
      innerIncludes)]
termPatterns VariableNode =
  [("variable", TmMatch (Just "variable.other.gotmpl") "\\$[a-zA-Z_]\\w*|\\$" [])]
termPatterns IdentifierNode =
  [("builtin", TmMatch (Just "support.function.gotmpl") (wordAlt builtinFunctions) [])]
termPatterns PipeNode = []

variableOpPatterns :: VariableOp -> [Named]
variableOpPatterns Declare =
  [("variable-declaration", TmMatch Nothing "(\\$[a-zA-Z_]\\w*)\\s*(:=)"
      [("1", "variable.other.gotmpl"), ("2", "keyword.operator.assignment.gotmpl")])]
variableOpPatterns Assign =
  [("variable-assignment", TmMatch Nothing "(\\$[a-zA-Z_]\\w*)\\s*(=)(?!=)"
      [("1", "variable.other.gotmpl"), ("2", "keyword.operator.assignment.gotmpl")])]

keywordEntry :: Named
keywordEntry = ("keyword", TmMatch (Just "keyword.control.gotmpl") (wordAlt keywords) [])

allEntries :: [Named]
allEntries = dedup [] $
  concatMap templateNodePatterns [minBound .. maxBound]
  ++ [keywordEntry]
  ++ concatMap actionBodyPatterns [minBound .. maxBound]
  ++ concatMap loopActionPatterns [minBound .. maxBound]
  ++ concatMap variableOpPatterns [minBound .. maxBound]
  ++ concatMap termPatterns [minBound .. maxBound]

dedup :: [String] -> [Named] -> [Named]
dedup _ [] = []
dedup seen ((k,p):rest)
  | k `elem` seen = dedup seen rest
  | otherwise = (k,p) : dedup (k:seen) rest

innerIncludes :: [TmPattern]
innerIncludes =
  let keys = [k | (k, _) <- allEntries, k /= "comment", k /= "action"]
  in map TmInclude keys

-- ==========================================================================
-- Main
-- ==========================================================================

main :: IO ()
main = do
  createDirectoryIfMissing True "syntaxes"
  let path = "syntaxes/gotemplate.tmLanguage.json"
  writeFile path (grammarJson allEntries)
  putStrLn $ "Generated: " ++ path
