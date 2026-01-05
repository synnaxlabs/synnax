// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type Synnax as Client } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import type * as monaco from "monaco-editor";
import { useEffect, useRef } from "react";

import { Lua } from "@/code/lua";
import { type UsePhantomGlobalsReturn } from "@/code/phantom";
import { useMonaco } from "@/code/Provider";

const ID = "onCommandSuggestionAccepted";

const suggestChannelNames = (
  mon: Pick<typeof monaco, "editor" | "KeyMod" | "KeyCode" | "KeyMod" | "languages">,
  onAccept: (channel: channel.Payload) => void,
  searcher?: channel.Client,
) => {
  const disposables: monaco.IDisposable[] = [];
  disposables.push(
    mon.editor.registerCommand(ID, (_, channel: channel.Payload) => {
      onAccept(channel);
    }),
  );
  disposables.push(
    mon.languages.registerCompletionItemProvider(Lua.LANGUAGE, {
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

        const channels = await searcher.retrieve({ searchTerm: word.word });
        const filteredChannels = IS_DEV
          ? channels
          : channels.filter(({ internal }) => !internal);
        return {
          suggestions: filteredChannels.map((channel) => ({
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
  useEffect(() => {
    if (monaco == null || client == null) return;
    disposables.current = suggestChannelNames(monaco, onAccept, client.channels);
    return () => disposables.current.forEach((d) => d.dispose());
  }, [monaco, client]);
};

export const bindChannelsAsGlobals = async (
  client: Client,
  prev: channel.Key[],
  current: channel.Key[],
  globals: UsePhantomGlobalsReturn,
) => {
  const removed = prev.filter((ch) => !current.includes(ch));
  removed.forEach((ch) => globals.del(ch.toString()));
  const added = current.filter((ch) => !prev.includes(ch));
  const channels = await client.channels.retrieve(added);
  channels.forEach((ch) => globals.set(ch.key.toString(), ch.name, ch.key.toString()));
};
