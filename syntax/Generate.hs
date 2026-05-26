{-# OPTIONS_GHC -Wall -Werror #-} -- ensures all cases are covered at compilation
module Main where

import System.Directory (createDirectoryIfMissing)
import qualified Data.ByteString.Lazy as LBS

import Grammar
    ( VariableOp(..),
      Term(..),
      LoopAction(..),
      ActionBody(..),
      TemplateNode(..),
      keywords,
      builtinFunctions )
import TextMate ( Named, TmPattern(..), TmSyntax(..), syntaxToJson)
import Regex ( wordAlt, dOpen, dClose
             , commentOpen, commentClose
             , pipe, comma, templateName
             , boolLiteral, numberLiteral
             , doubleQuote, stringEscape, backtick
             , charLiteral, nilLiteral, dot, field
             , parenOpen, parenClose
             , variable, varDeclare, varAssign )

-- regex specifications for all grammar elements
-- scopes are taken from 2026-dark.json VSCode theme

templateNodePatterns :: TemplateNode -> [Named]
templateNodePatterns TextNode = []
templateNodePatterns CommentNode =
  [("comment", TmRegion (Just "comment.block.gotmpl")
      commentOpen commentClose
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
  [("pipe", TmMatch (Just "keyword.operator.pipe.gotmpl") pipe [])]
actionBodyPatterns IfAction = []
actionBodyPatterns RangeAction =
  [("comma", TmMatch (Just "punctuation.separator.comma.gotmpl") comma [])]
actionBodyPatterns WithAction = []
actionBodyPatterns TemplateInvokeAction =
  [("template-name", TmMatch Nothing templateName
      [("1", "entity.name.function.gotmpl")])]
actionBodyPatterns BlockAction = []  -- name covered by template-name lookbehind
actionBodyPatterns DefineAction = [] -- name covered by template-name lookbehind

loopActionPatterns :: LoopAction -> [Named]
loopActionPatterns BreakAction = []    -- covered by keywords list
loopActionPatterns ContinueAction = [] -- covered by keywords list


termPatterns :: Term -> [Named]
termPatterns BoolLiteral =
  [("boolean", TmMatch (Just "constant.language.boolean.gotmpl") boolLiteral [])]
termPatterns NumberLiteral =
  [("number", TmMatch (Just "constant.numeric.gotmpl") numberLiteral [])]
termPatterns StringLiteral =
  [("string-double", TmRegion (Just "string.quoted.double.gotmpl")
      doubleQuote doubleQuote
      [("0", "punctuation.definition.string.begin.gotmpl")]
      [("0", "punctuation.definition.string.end.gotmpl")]
      [TmMatch (Just "constant.character.escape.gotmpl") stringEscape []])
  ,("string-raw", TmRegion (Just "string.quoted.other.raw.gotmpl")
      backtick backtick
      [("0", "punctuation.definition.string.begin.gotmpl")]
      [("0", "punctuation.definition.string.end.gotmpl")]
      [])]
termPatterns CharLiteral =
  [("char", TmMatch (Just "string.quoted.single.gotmpl") charLiteral [])]
termPatterns NilLiteral =
  [("nil", TmMatch (Just "constant.language.nil.gotmpl") nilLiteral [])]
termPatterns DotLiteral =
  [("dot", TmMatch (Just "variable.language.dot.gotmpl") dot [])]
termPatterns FieldNode =
  [("field", TmMatch (Just "variable.other.member.gotmpl") field [])]
termPatterns ChainNode =
  [("parentheses", TmRegion Nothing
      parenOpen parenClose
      [("0", "punctuation.section.parens.begin.gotmpl")]
      [("0", "punctuation.section.parens.end.gotmpl")]
      innerIncludes)]
termPatterns VariableNode =
  [("variable", TmMatch (Just "variable.other.gotmpl") variable [])]
termPatterns IdentifierNode =
  [("builtin", TmMatch (Just "support.function.gotmpl") (wordAlt builtinFunctions) [])]
termPatterns PipeNode = []

variableOpPatterns :: VariableOp -> [Named]
variableOpPatterns Declare =
  [("variable-declaration", TmMatch Nothing varDeclare
      [("1", "variable.other.gotmpl"), ("2", "keyword.operator.assignment.gotmpl")])]
variableOpPatterns Assign =
  [("variable-assignment", TmMatch Nothing varAssign
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

syntax :: TmSyntax
syntax = TmSyntax
  "source.gotmpl"
  ["tmpl", "gotmpl"]
  []
  [TmInclude "comment", TmInclude "action"]
  allEntries

-- ==========================================================================
-- Main
-- ==========================================================================

main :: IO ()
main = do
  createDirectoryIfMissing True "syntaxes"
  let path   = "syntaxes/gotemplate.tmLanguage.json"
  LBS.writeFile path (syntaxToJson syntax)
  putStrLn $ "Generated: " ++ path
