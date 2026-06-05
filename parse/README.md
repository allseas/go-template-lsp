# Parser

This is an extended version of the go/text/template/parse package.

Changes implemented:

- added new node type (Undefined)
- added new operation mode to the parser (ParsePartial)
- added non-breaking error handling in ParsePartial mode
- added tests enforcing tree structure on malformed input
- added new field to Tree: Errors[]

- although some execution paths on the default mode have been slightly modified, they only get reached in case of incorrect input. And still return a nil tree, as well as the same error message as before. Thus not violating backwards compatibility.

- any malformed input is constrained to remain a local undefined node to the best of our ability, i.e. a syntax error inside a pipeline will remain inside a single command, instead of corrupting the whole pipeline.

- The lexer has been modified to detect a left delimiter inside an already open action, which it didnt do before.
- The lexer response to ':' has been changed. Now if its not followed by '=' it simply returns the unicode char, instead of consuming the following character and producing an error.

- Fixed `FieldNode` position for chained field accesses (e.g. `.Address.Country`). Previously the node's `Pos` pointed to the second field in the chain rather than the first, because the `FieldNode` used to anchor at the next peeked token's position. The `FieldNode` is now anchored at the original node's position, so the resulting `FieldNode.Pos` correctly points to the leading `.` of the first field.

We aim to have these changes eventually merged into the upstream of go, then having this package locally will become obsolete.
