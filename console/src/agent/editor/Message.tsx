// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { grammar } from "@synnaxlabs/arc";
import { type agent } from "@synnaxlabs/client";
import { Button, Flex, Icon, Text, Theming } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useState } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { type HighlighterCore } from "shiki";
import { createHighlighterCore } from "shiki/core";
import { createOnigurumaEngine } from "shiki/engine/oniguruma";

import { CSS } from "@/css";

let highlighterPromise: Promise<HighlighterCore> | null = null;

const getHighlighter = (): Promise<HighlighterCore> => {
  if (highlighterPromise != null) return highlighterPromise;
  highlighterPromise = createHighlighterCore({
    engine: createOnigurumaEngine(import("shiki/wasm")),
    themes: [import("shiki/themes/github-dark"), import("shiki/themes/github-light")],
    langs: [grammar],
  });
  return highlighterPromise;
};

interface MessageProps {
  message: agent.Message;
}

const useHighlightedCode = (code: string, lang: string): string | null => {
  const [html, setHtml] = useState<string | null>(null);
  const theme = Theming.use();
  const isDark = theme.key.includes("Dark");

  useEffect(() => {
    if (lang !== "arc") {
      setHtml(null);
      return;
    }
    let cancelled = false;
    getHighlighter()
      .then((highlighter) => {
        if (cancelled) return;
        const result = highlighter.codeToHtml(code, {
          lang: "arc",
          theme: isDark ? "github-dark" : "github-light",
        });
        setHtml(result);
      })
      .catch(() => setHtml(null));
    return () => {
      cancelled = true;
    };
  }, [code, lang, isDark]);

  return html;
};

const CodeBlock = ({
  className,
  children,
}: {
  className?: string;
  children?: string;
}): ReactElement => {
  const match = /language-(\w+)/.exec(className ?? "");
  const lang = match?.[1] ?? "";
  const code = (children ?? "").replace(/\n$/, "");
  const highlightedHtml = useHighlightedCode(code, lang);
  const [expanded, setExpanded] = useState(false);

  return (
    <div
      className={CSS(
        CSS.BE("agent-editor", "code-block"),
        expanded && CSS.BEM("agent-editor", "code-block", "expanded"),
      )}
    >
      <div
        className={CSS.BE("agent-editor", "code-header")}
        onClick={() => setExpanded((prev) => !prev)}
      >
        <Text.Text level="small" color={8}>
          {expanded ? "Hide" : "Show"} Arc code
        </Text.Text>
        <Button.Copy
          text={code}
          variant="text"
          size="small"
          style={{ visibility: expanded ? "visible" : "hidden" }}
        />
        <Icon.Caret.Right
          className={CSS(
            CSS.BE("agent-editor", "code-caret"),
            expanded && CSS.BEM("agent-editor", "code-caret", "expanded"),
          )}
        />
      </div>
      {expanded &&
        (highlightedHtml != null ? (
          <div
            className={CSS.BE("agent-editor", "code-pre")}
            dangerouslySetInnerHTML={{ __html: highlightedHtml }}
          />
        ) : (
          <pre className={CSS.BE("agent-editor", "code-pre")}>
            <code>{code}</code>
          </pre>
        ))}
    </div>
  );
};

const markdownComponents = {
  code: ({
    className,
    children,
  }: {
    className?: string;
    children?: React.ReactNode;
  }) => {
    const isBlock = /language-/.test(className ?? "");
    if (isBlock)
      return (
        <CodeBlock className={className}>{children as string}</CodeBlock>
      );
    return <code className={className}>{children}</code>;
  },
};

export const Message = ({ message }: MessageProps): ReactElement => {
  const isAgent = message.role !== "user";
  const roleClass = isAgent ? "agent" : "user";

  return (
    <div
      className={CSS(
        CSS.BE("agent-editor", "message"),
        CSS.BEM("agent-editor", "message", roleClass),
      )}
    >
      <div
        className={CSS(
          CSS.BE("agent-editor", "avatar"),
          CSS.BEM("agent-editor", "avatar", roleClass),
        )}
      >
        {isAgent ? <Icon.Auto /> : <Icon.User />}
      </div>
      <div className={CSS.BE("agent-editor", "body")}>
        <Text.Text
          level="small"
          weight={500}
          color={8}
          className={CSS.BE("agent-editor", "role")}
        >
          {isAgent ? "Agent" : "You"}
        </Text.Text>
        <div className={CSS.BE("agent-editor", "content")}>
          {isAgent ? (
            <Markdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
              {message.content}
            </Markdown>
          ) : (
            <Text.Text level="p" style={{ whiteSpace: "pre-wrap" }}>
              {message.content}
            </Text.Text>
          )}
        </div>
        {isAgent && (
          <div className={CSS.BE("agent-editor", "message-actions")}>
            <Button.Copy text={message.content} variant="text" size="small" />
          </div>
        )}
      </div>
    </div>
  );
};

export const LoadingMessage = (): ReactElement => (
  <div
    className={CSS(
      CSS.BE("agent-editor", "message"),
      CSS.BEM("agent-editor", "message", "agent"),
    )}
  >
    <div
      className={CSS(
        CSS.BE("agent-editor", "avatar"),
        CSS.BEM("agent-editor", "avatar", "agent"),
      )}
    >
      <Icon.Auto />
    </div>
    <div className={CSS.BE("agent-editor", "body")}>
      <Flex.Box x align="center" gap="small">
        <Icon.Loading />
        <Text.Text level="p" color={7}>
          Thinking...
        </Text.Text>
      </Flex.Box>
    </div>
  </div>
);
