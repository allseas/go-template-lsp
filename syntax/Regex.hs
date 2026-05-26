{-# OPTIONS_GHC -Wall -Werror #-}
-- this file contains all regexes used for go template
module Regex where

import Data.List (intercalate)

-- escape regex special characters
escRegex :: String -> String
escRegex = concatMap (\c -> if c `elem` ("{}()[]\\.*+?^$|-" :: String) then ['\\',c] else [c])

-- disjunction
wordAlt :: [String] -> String
wordAlt ws = "\\b(" ++ intercalate "|" ws ++ ")\\b"
-- default template open delimiter
dOpen :: String
dOpen = escRegex "{{-" ++ "?"
-- default template close
dClose :: String
dClose = escRegex "-" ++ "?" ++ escRegex "}}"

commentOpen :: String
commentOpen = dOpen ++ "\\s*/\\*"

commentClose :: String
commentClose = "\\*/\\s*" ++ dClose

pipe :: String
pipe = "\\|"

comma :: String
comma = ","

templateName :: String
templateName = "(?<=template|block|define)\\s+(\"[^\"]*\")"

-- template documentation specifies "Go syntax" for all literals, this is made following the go spec

decimalDigits :: String
decimalDigits = "[0-9](?:_?[0-9])*"

binaryDigits :: String
binaryDigits = "[01](?:_?[01])*"

octalDigits :: String
octalDigits = "[0-7](?:_?[0-7])*"

hexDigits :: String
hexDigits = "[0-9A-Fa-f](?:_?[0-9A-Fa-f])*"

decimalExponent :: String
decimalExponent = "[eE][-+]?" ++ decimalDigits

hexExponent :: String
hexExponent = "[pP][-+]?" ++ decimalDigits

decimalLiteral :: String
decimalLiteral = "(?:0|[1-9](?:_?[0-9](?:_?[0-9])*)?)"

binaryLiteral :: String
binaryLiteral = "0[bB]_?" ++ binaryDigits

octalLiteral :: String
octalLiteral = "0[oO]?_?" ++ octalDigits

hexLiteral :: String
hexLiteral = "0[xX]_?" ++ hexDigits

integerLiteral :: String
integerLiteral = "(?:" ++ hexLiteral ++ "|" ++ binaryLiteral ++ "|" ++ octalLiteral ++ "|" ++ decimalLiteral ++ ")"

hexMantissa :: String
hexMantissa = "(?:_?" ++ hexDigits ++ "(?:\\.(?:" ++ hexDigits ++ ")?)?|\\." ++ hexDigits ++ ")"

hexFloatLiteral :: String
hexFloatLiteral = "0[xX]" ++ hexMantissa ++ hexExponent

decimalFloatLiteral :: String
decimalFloatLiteral = decimalDigits ++ "(?:\\.(?:" ++ decimalDigits ++ ")?(?:" ++ decimalExponent ++ ")?|" ++ decimalExponent ++ ")"

floatLiteral :: String
floatLiteral = "(?:" ++ hexFloatLiteral ++ "|" ++ decimalFloatLiteral ++ ")"

boolLiteral :: String
boolLiteral = "\\b(true|false)\\b"

numberLiteral :: String
numberLiteral = "\\b(?:" ++ floatLiteral ++ "|" ++ integerLiteral ++ ")i?\\b"

doubleQuote :: String
doubleQuote = "\""

stringEscape :: String
stringEscape = "\\\\(?:[abfnrtv\\\\\"']|x[0-9A-Fa-f]{2}|[0-7]{3}|u[0-9A-Fa-f]{4}|U[0-9A-Fa-f]{8})"

backtick :: String
backtick = "`"

charLiteral :: String
charLiteral = "'(?:" ++ stringEscape ++ "|[^'\\\\])'"

nilLiteral :: String
nilLiteral = "\\bnil\\b"

dot :: String
dot = "(?<![a-zA-Z_\\w])\\.(?![a-zA-Z_\\w])"

field :: String
field = "\\.[a-zA-Z_]\\w*"

parenOpen :: String
parenOpen = "\\("

parenClose :: String
parenClose = "\\)"

variable :: String
variable = "\\$[a-zA-Z_]\\w*|\\$"

varDeclare :: String
varDeclare = "(\\$[a-zA-Z_]\\w*)\\s*(:=)"

varAssign :: String
varAssign = "(\\$[a-zA-Z_]\\w*)\\s*(=)"