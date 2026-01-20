// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package formatter

import (
	"github.com/antlr4-go/antlr/v4"
)

type commentAttacher struct {
	comments []antlr.Token
	used     []bool
}

func newCommentAttacher(comments []antlr.Token) *commentAttacher {
	return &commentAttacher{
		comments: comments,
		used:     make([]bool, len(comments)),
	}
}

func (ca *commentAttacher) getLeadingComments(tok antlr.Token) []antlr.Token {
	if tok == nil {
		return nil
	}
	tokLine := tok.GetLine()
	var leading []antlr.Token

	for i, comment := range ca.comments {
		if ca.used[i] {
			continue
		}
		commentLine := comment.GetLine()
		if commentLine < tokLine {
			leading = append(leading, comment)
			ca.used[i] = true
		}
	}
	return leading
}

func (ca *commentAttacher) getTrailingComment(tok antlr.Token) antlr.Token {
	if tok == nil {
		return nil
	}
	tokLine := tok.GetLine()
	tokEnd := tok.GetStop()

	for i, comment := range ca.comments {
		if ca.used[i] {
			continue
		}
		commentLine := comment.GetLine()
		commentStart := comment.GetStart()
		if commentLine == tokLine && commentStart > tokEnd {
			ca.used[i] = true
			return comment
		}
	}
	return nil
}

func (ca *commentAttacher) getRemainingComments() []antlr.Token {
	var remaining []antlr.Token
	for i, comment := range ca.comments {
		if !ca.used[i] {
			remaining = append(remaining, comment)
			ca.used[i] = true
		}
	}
	return remaining
}
