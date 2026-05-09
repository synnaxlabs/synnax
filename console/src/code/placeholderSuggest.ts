// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// monaco-vscode-api ignores `tokenTypes` in language config, so the parent
// `string.quoted.raw.arc` scope suppresses the popup inside `{...}`. The only
// alternative is `quickSuggestions.strings: "on"`, which fires in every string.
export const PLACEHOLDER_RE = /`[^`]*\{[^}]*$/;
const WORD_CHAR_RE = /^\w$/;

export const shouldTriggerSuggestion = (
  bufferBeforeCaret: string,
  typedChar: string,
): boolean => WORD_CHAR_RE.test(typedChar) && PLACEHOLDER_RE.test(bufferBeforeCaret);
