{-# OPTIONS_GHC -Wall -Werror #-}
module Grammar where
-- Context Free Grammar for text/template
data TemplateNode = TextNode | CommentNode | ActionNode
  deriving (Enum, Bounded)

data ActionBody
  = PipelineAction
  | IfAction
  | RangeAction
  | WithAction
  | TemplateInvokeAction
  | BlockAction
  | DefineAction
  deriving (Enum, Bounded)
-- Enum and Bounded let us iterate over all constructors
data LoopAction = BreakAction | ContinueAction
  deriving (Enum, Bounded)

data Term
  = BoolLiteral
  | NumberLiteral
  | StringLiteral
  | CharLiteral
  | NilLiteral
  | DotLiteral
  | FieldNode
  | ChainNode
  | VariableNode
  | IdentifierNode
  | PipeNode
  deriving (Enum, Bounded)

data VariableOp = Declare | Assign
  deriving (Enum, Bounded)

-- string syntax constants used in regexes

keywords :: [String]
keywords =
  ["if", "else", "end", "range", "with",
   "template", "block", "define", "break", "continue"]

builtinFunctions :: [String]
builtinFunctions =
  ["and", "call", "html", "index", "slice", "js", "len", "not", "or",
   "print", "printf", "println", "urlquery",
   "eq", "ne", "lt", "le", "gt", "ge"]
