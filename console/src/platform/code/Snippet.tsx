// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/code/Snippet.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Fragment, type ReactElement } from "react";

import { CSS } from "@/platform/css";

/** A run of code, colored by its kind when it has one. */
export type Token =
  string | { kind: "keyword" | "string" | "function" | "comment"; text: string };

/** Lines of tokens. One source gives both the colored code and the copied text. */
export type Lines = Token[][];

export const keyword = (text: string): Token => ({ kind: "keyword", text });
/** A string literal. Wraps text in double quotes. */
export const quoted = (text: string): Token => ({ kind: "string", text: `"${text}"` });
/** The name of a function or command that the code calls. */
export const call = (text: string): Token => ({ kind: "function", text });
export const comment = (text: string): Token => ({ kind: "comment", text });

const toText = (code: Lines): string =>
  code
    .map((line) => line.map((t) => (typeof t === "string" ? t : t.text)).join(""))
    .join("\n");

export interface SnippetProps {
  /** What the snippet holds, as assistive technology names it. */
  label: string;
  code: Lines;
  /** The code to copy, when it differs from what the snippet shows. */
  copy?: Lines;
}

/** Shows colored code with a button that copies it. */
export const Snippet = ({ label, code, copy = code }: SnippetProps): ReactElement => (
  <Flex.Box x align="start" className={CSS.B("code-snippet")}>
    <pre aria-label={label}>
      {code.map((line, i) => (
        <Fragment key={i}>
          {i > 0 && "\n"}
          {line.map((token, j) =>
            typeof token === "string" ? (
              token
            ) : (
              <span key={j} className={CSS.BEM("code-snippet", "token", token.kind)}>
                {token.text}
              </span>
            ),
          )}
        </Fragment>
      ))}
    </pre>
    <Button.Copy
      text={toText(copy)}
      variant="text"
      size="small"
      textColor={9}
      aria-label={`Copy ${label}`}
    />
  </Flex.Box>
);
