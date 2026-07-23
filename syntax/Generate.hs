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
  []

-- Host languages embedded inside TextNode based on the compound file extension.
-- Each entry: (extension key, host scope name).
-- A file named foo.<key>.tmpl gets highlighting for that host language between
-- go-template actions/comments.
hostLanguages :: [(String, String)]
hostLanguages =
  [ ("sql",  "source.sql")
  , ("html", "text.html.basic")
  , ("json", "source.json")
  , ("yaml", "source.yaml")
  , ("css",  "source.css")
  , ("js",   "source.js")
  , ("xml",  "text.xml")
  , ("md",   "text.html.markdown")
  , ("sh",   "source.shell")
  , ("scl",  "source.scl")
  , ("cpp", "source.cpp")
  ]

-- Derived grammar for a host language. Falls through to the host language
-- grammar as the base tokenisation, and uses an in-grammar 'injections' rule
-- to contribute go-template action/comment patterns at every nesting level of
-- the host grammar (including inside its begin/end regions and strings) --
-- not only at the top level. The in-grammar 'injections' form (as opposed to
-- a separate injection grammar with 'injectionSelector') works in both VS
-- Code (vscode-textmate) and JetBrains (only the in-grammar form is
-- recognised by the JetBrains TextMate parser -- it reads only the
-- 'injections' key, not top-level 'injectionSelector').
--
-- The selector excludes meta.embedded.gotmpl (already inside an action) and
-- comment.block.gotmpl (already inside a template comment) to avoid recursing
-- into ourselves. The 'L:' prefix gives the injection higher priority than
-- the host grammar's patterns.
derivedSyntax :: (String, String) -> TmSyntax
derivedSyntax (key, hostScope) = TmSyntax
  ("source.gotmpl." ++ key)
  [key ++ ".tmpl", key ++ ".gotmpl"]
  []
  [ TmIncludeScope hostScope ]
  []
  [ ( "L:source.gotmpl." ++ key
        ++ " - meta.embedded.gotmpl - comment.block.gotmpl"
    , [ TmIncludeScope "source.gotmpl#comment"
      , TmIncludeScope "source.gotmpl#action"
      ]
    )
  ]

-- ==========================================================================
-- Main
-- ==========================================================================

main :: IO ()
main = do
  createDirectoryIfMissing True "syntaxes"
  let basePath = "syntaxes/gotemplate.tmLanguage.json"
  LBS.writeFile basePath (syntaxToJson syntax)
  putStrLn $ "Generated: " ++ basePath
  mapM_ writeDerived hostLanguages
  where
    writeDerived entry@(key, _) = do
      let path = "syntaxes/gotemplate-" ++ key ++ ".tmLanguage.json"
      LBS.writeFile path (syntaxToJson (derivedSyntax entry))
      putStrLn $ "Generated: " ++ path
