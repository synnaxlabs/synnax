// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { type AsyncTermSearcher } from "@synnaxlabs/x";
import type * as monaco from "monaco-editor";
import { useRef } from "react";

import { useMonaco } from "@/code/Provider";

const ID = "onCommandSuggestionAccepted";

const suggestChannelNames = (
  mon: Pick<typeof monaco, "editor" | "KeyMod" | "KeyCode" | "KeyMod" | "languages">,
  onAccept: (channel: channel.Payload) => void,
  searcher?: AsyncTermSearcher<string, channel.Key, channel.Payload>,
) => {
  const disposables: monaco.IDisposable[] = [];
  disposables.push(
    mon.editor.registerCommand(ID, (_, channel: channel.Payload) => {
      onAccept(channel);
    }),
  );
  disposables.push(
    mon.languages.registerCompletionItemProvider("lua", {
      triggerCharacters: ["."],
      provideCompletionItems: async (
        model: monaco.editor.ITextModel,
        position: monaco.Position,
      ): Promise<monaco.languages.CompletionList> => {
        if (searcher == null) return { suggestions: [] };
        const word = model.getWordUntilPosition(position);
        const range: monaco.IRange = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn,
        };

        // Check if we're in a set() call
        const lineContent = model.getLineContent(position.lineNumber);
        const beforeWord = lineContent.substring(0, word.startColumn - 1);
        const isInSetCall = /set\s*\($/.test(beforeWord.trim());

        const channels = await searcher.search(word.word);
        return {
          suggestions: channels.map((channel) => ({
            label: channel.name,
            kind: mon.languages.CompletionItemKind.Variable,
            insertText: isInSetCall
              ? `"${channel.name}"`
              : channel.name.includes("-")
                ? `get("${channel.name}")`
                : channel.name,
            range,
            command: {
              id: ID,
              title: "Suggestion Accepted",
              arguments: [channel],
            },
          })),
        };
      },
    }),
  );
  return disposables;
};

export const useSuggestChannels = (onAccept: (channel: channel.Payload) => void) => {
  const monaco = useMonaco();
  const client = Synnax.use();
  const disposables = useRef<monaco.IDisposable[]>([]);
  useAsyncEffect(async () => {
    if (monaco == null || client == null) return;
    disposables.current = suggestChannelNames(monaco, onAccept, client.channels);
    return () => disposables.current.forEach((d) => d.dispose());
  }, [monaco, client]);
};
