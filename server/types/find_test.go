package types

import "testing"

func TestNodeFind(t *testing.T) {
	for _, tc := range nodeFindTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := NodeFind(tc.root, tc.offset)
			if got != tc.wantNode {
				if tc.wantNode == nil {
					t.Fatalf("expected nil, got %T at pos %d", got, got.Position())
				}
				if got == nil {
					t.Fatalf("expected %T at pos %d, got nil", tc.wantNode, tc.wantNode.Position())
				}
				t.Fatalf("expected %T at pos %d, got %T at pos %d",
					tc.wantNode, tc.wantNode.Position(),
					got, got.Position(),
				)
			}
		})
	}
}

func TestEnclosingList(t *testing.T) {
	for _, tc := range enclosingListTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := EnclosingList(tc.node)
			if got != tc.wantList {
				t.Fatalf("expected %v, got %v", tc.wantList, got)
			}
		})
	}
}

func TestEnclosingPipe(t *testing.T) {
	for _, tc := range enclosingPipeTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := EnclosingPipe(tc.node)
			if got != tc.wantPipe {
				t.Fatalf("expected %v, got %v", tc.wantPipe, got)
			}
		})
	}
}

func TestEnclosingCommand(t *testing.T) {
	for _, tc := range enclosingCommandTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := EnclosingCommand(tc.node)
			if got != tc.wantCmd {
				t.Fatalf("expected %v, got %v", tc.wantCmd, got)
			}
		})
	}
}
