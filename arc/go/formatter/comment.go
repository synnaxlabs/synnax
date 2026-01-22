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
	comments   []antlr.Token
	nextUnused int
}

func newCommentAttacher(comments []antlr.Token) *commentAttacher {
	return &commentAttacher{comments: comments}
}

func (ca *commentAttacher) getLeadingComments(tok antlr.Token) []antlr.Token {
	if tok == nil || ca.nextUnused >= len(ca.comments) {
		return nil
	}
	tokLine := tok.GetLine()
	var leading []antlr.Token

	for ca.nextUnused < len(ca.comments) {
		comment := ca.comments[ca.nextUnused]
		if comment.GetLine() >= tokLine {
			break
		}
		leading = append(leading, comment)
		ca.nextUnused++
	}
	return leading
}

func (ca *commentAttacher) getTrailingComment(tok antlr.Token) antlr.Token {
	if tok == nil || ca.nextUnused >= len(ca.comments) {
		return nil
	}
	tokLine := tok.GetLine()
	tokEnd := tok.GetStop()

	comment := ca.comments[ca.nextUnused]
	commentLine := comment.GetLine()
	commentStart := comment.GetStart()
	if commentLine == tokLine && commentStart > tokEnd {
		ca.nextUnused++
		return comment
	}
	return nil
}

func (ca *commentAttacher) getRemainingComments() []antlr.Token {
	if ca.nextUnused >= len(ca.comments) {
		return nil
	}
	remaining := ca.comments[ca.nextUnused:]
	ca.nextUnused = len(ca.comments)
	return remaining
}
