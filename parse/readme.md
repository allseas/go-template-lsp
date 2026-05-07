## Parser
This is an extended version of the go/test/template/parse package.

Changes implemented:

- added new node type (Undefined)
- added new operation mode to the parser (IgnoreErrors)
- added non-breaking error handling in ignore errors mode
- added some tests for the ignore errrors mode

We aim to have these changes eventually merged into the upstream of go, then having this package locally will become obsolete.