// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ExtensionHostKind,
  registerExtension,
} from "@codingame/monaco-vscode-api/extensions";
import { type arc, type Synnax } from "@synnaxlabs/client";
import { type Stream } from "@synnaxlabs/freighter";
import { type destructor } from "@synnaxlabs/x";
import { MonacoLanguageClient } from "monaco-languageclient";
import { type Message, type MessageReader, type MessageWriter } from "vscode-jsonrpc";
import { CloseAction, ErrorAction } from "vscode-languageclient/browser";

import arcGrammar from "@/code/arc/arc.tmLanguage.json";
import arcLanguageConfiguration from "@/code/arc/language-configuration.json";
import { type Extension } from "@/code/init/initialize";

const NOOP_DISPOSER = () => ({ dispose: () => {} });

export const LANGUAGE = "arc";

interface FreighterTransportProps {
  stream: Stream<typeof arc.lspMessageZ, typeof arc.lspMessageZ>;
}

const createFreighterTransport = ({
  stream,
}: FreighterTransportProps): {
  reader: MessageReader;
  writer: MessageWriter;
} => {
  let isClosed = false;
  let onCloseCallback: (() => void) | null = null;
  let onErrorCallback: ((error: Error) => void) | null = null;

  // Start receiving messages in the background
  const receiveLoop = async () => {
    try {
      while (!isClosed) {
        const [msg, err] = await stream.receive();
        if (err != null) {
          if (onErrorCallback != null) onErrorCallback(err);
          break;
        }
        if (msg == null) break;

        // Parse the raw JSON message
        try {
          const parsed = JSON.parse(msg.content);
          if (onMessageCallback != null) onMessageCallback(parsed);
        } catch (parseError) {
          if (onErrorCallback != null)
            onErrorCallback(
              parseError instanceof Error ? parseError : new Error(String(parseError)),
            );
        }
      }
    } finally {
      isClosed = true;
      if (onCloseCallback != null) onCloseCallback();
    }
  };

  let onMessageCallback: ((message: Message) => void) | null = null;

  const reader: MessageReader = {
    listen: (callback) => {
      onMessageCallback = callback as (message: Message) => void;
      receiveLoop().catch((err) => {
        if (onErrorCallback != null) onErrorCallback(err);
      });
      return {
        dispose: () => {
          onMessageCallback = null;
        },
      };
    },
    dispose: () => {
      isClosed = true;
    },
    onError: (callback) => {
      onErrorCallback = callback;
      return {
        dispose: () => {
          onErrorCallback = null;
        },
      };
    },
    onClose: (callback) => {
      onCloseCallback = callback;
      return {
        dispose: () => {
          onCloseCallback = null;
        },
      };
    },
    onPartialMessage: NOOP_DISPOSER,
  };

  const writer: MessageWriter = {
    write: async (message) => {
      if (isClosed) throw new Error("Stream is closed");
      stream.send({ content: JSON.stringify(message) });
    },
    dispose: () => {
      isClosed = true;
    },
    onError: (callback) => {
      const wrappedCallback = (err: Error) => callback([err, undefined, undefined]);
      onErrorCallback = wrappedCallback;
      return {
        dispose: () => {
          onErrorCallback = null;
        },
      };
    },
    onClose: (callback) => {
      onCloseCallback = callback;
      return {
        dispose: () => {
          onCloseCallback = null;
        },
      };
    },
    end: () => {
      isClosed = true;
      stream.closeSend();
    },
  };

  return { reader, writer };
};

let synnaxClient: Synnax | null = null;

export const setSynnaxClient = (client: Synnax) => {
  synnaxClient = client;
};

const startArcLSP = async (): Promise<destructor.Async> => {
  if (synnaxClient == null) {
    console.warn("Synnax client not set, Arc LSP will not start");
    return async () => {};
  }

  try {
    const stream = await synnaxClient.arcs.openLSP();

    const { reader, writer } = createFreighterTransport({ stream });

    const languageClient = new MonacoLanguageClient({
      name: "Arc Language Server",
      clientOptions: {
        documentSelector: [LANGUAGE],
        errorHandler: {
          error: () => ({ action: ErrorAction.Continue }),
          closed: () => ({ action: CloseAction.DoNotRestart }),
        },
      },
      messageTransports: { reader, writer },
    });

    await languageClient.start();

    return async () => {
      await languageClient.stop();
      stream.closeSend();
    };
  } catch (error) {
    console.error("Failed to start Arc LSP:", error);
    return async () => {};
  }
};

export type SemanticTokenType =
  | "type"
  | "function"
  | "parameter"
  | "variable"
  | "keyword"
  | "string"
  | "number"
  | "comment"
  | "operator"
  | "channel"
  | "sequence"
  | "stage"
  | "block"
  | "statefulVariable"
  | "edgeOneShot"
  | "edgeContinuous"
  | "constant"
  | "config"
  | "input"
  | "output"
  | "unit";

export type SemanticTokenColors = Record<SemanticTokenType, string>;

export interface ThemedSemanticTokenColors {
  dark: SemanticTokenColors;
  light: SemanticTokenColors;
}

export const ARC_SEMANTIC_TOKEN_COLORS: ThemedSemanticTokenColors = {
  dark: {
    statefulVariable: "#E5A84B",
    edgeOneShot: "#E06C75",
    edgeContinuous: "#56c8d8",
    channel: "#61AFEF",
    keyword: "#CC255F",
    type: "#4EC9B0",
    string: "#98C379",
    number: "#98C379",
    variable: "#dadada",
    function: "#556bf8",
    sequence: "#dadada",
    stage: "#dadada",
    block: "#dadada",
    parameter: "#dadada",
    config: "#dadada",
    input: "#dadada",
    output: "#dadada",
    constant: "#dadada",
    operator: "#dadada",
    unit: "#dadada",
    comment: "#5C6370",
  },
  light: {
    statefulVariable: "#B45000",
    edgeOneShot: "#BE3E4A",
    edgeContinuous: "#0097A7",
    channel: "#0070C1",
    keyword: "#CC255F",
    type: "#267F99",
    string: "#0A7D00",
    number: "#0A7D00",
    variable: "#292929",
    function: "#3774D0",
    sequence: "#292929",
    stage: "#292929",
    block: "#292929",
    parameter: "#292929",
    config: "#292929",
    input: "#292929",
    output: "#292929",
    constant: "#292929",
    operator: "#292929",
    unit: "#292929",
    comment: "#9DA5B4",
  },
};

const GRAMMAR_PATH = "./arc.tmLanguage.json";
const LANGUAGE_CONFIGURATION_PATH = "./language-configuration.json";

const registerArcLanguage = async (): Promise<destructor.Async> => {
  const { registerFileUrl } = registerExtension(
    {
      name: "arc-language",
      publisher: "synnaxlabs",
      version: "1.0.0",
      engines: { vscode: "*" },
      contributes: {
        languages: [
          {
            id: LANGUAGE,
            aliases: ["Arc", "arc"],
            extensions: [".arc"],
            configuration: LANGUAGE_CONFIGURATION_PATH,
          },
        ],
        grammars: [
          {
            language: LANGUAGE,
            scopeName: "source.arc",
            path: GRAMMAR_PATH,
          },
        ],
      },
    },
    ExtensionHostKind.LocalProcess,
  );

  registerFileUrl(
    GRAMMAR_PATH,
    `data:application/json;base64,${btoa(JSON.stringify(arcGrammar))}`,
  );
  registerFileUrl(
    LANGUAGE_CONFIGURATION_PATH,
    `data:application/json;base64,${btoa(JSON.stringify(arcLanguageConfiguration))}`,
  );

  return async () => {};
};

// TextMate scope to color mappings (for hover popups and fallback highlighting)
const ARC_TEXTMATE_RULES = {
  dark: [
    { scope: "keyword.control.arc", settings: { foreground: "#CC255F" } },
    { scope: "keyword.other.arc", settings: { foreground: "#CC255F" } },
    { scope: "keyword.operator.logical.arc", settings: { foreground: "#CC255F" } },
    { scope: "keyword.operator.arithmetic.arc", settings: { foreground: "#dadada" } },
    { scope: "keyword.operator.comparison.arc", settings: { foreground: "#dadada" } },
    { scope: "keyword.operator.assignment.arc", settings: { foreground: "#dadada" } },
    {
      scope: "keyword.operator.assignment.declare.arc",
      settings: { foreground: "#dadada" },
    },
    {
      scope: "keyword.operator.assignment.stateful.arc",
      settings: { foreground: "#E5A84B" },
    },
    { scope: "keyword.operator.channel.arc", settings: { foreground: "#dadada" } },
    { scope: "keyword.operator.transition.arc", settings: { foreground: "#E06C75" } },
    { scope: "keyword.operator.flow.arc", settings: { foreground: "#56c8d8" } },
    { scope: "constant.language.boolean.arc", settings: { foreground: "#CC255F" } },
    { scope: "constant.language.null.arc", settings: { foreground: "#CC255F" } },
    { scope: "string.quoted.double.arc", settings: { foreground: "#98C379" } },
    { scope: "string.quoted.single.arc", settings: { foreground: "#98C379" } },
    { scope: "constant.numeric", settings: { foreground: "#98C379" } },
    { scope: "support.type.primitive.arc", settings: { foreground: "#4EC9B0" } },
    { scope: "support.type.composite.arc", settings: { foreground: "#4EC9B0" } },
    { scope: "support.type.channel.arc", settings: { foreground: "#61AFEF" } },
    { scope: "comment", settings: { foreground: "#5C6370" } },
    { scope: "entity.name.function.arc", settings: { foreground: "#556bf8" } },
    { scope: "support.function.builtin.arc", settings: { foreground: "#556bf8" } },
    {
      scope: "support.function.builtin.stage.arc",
      settings: { foreground: "#dadada" },
    },
    { scope: "entity.name.type.sequence.arc", settings: { foreground: "#dadada" } },
    { scope: "entity.name.type.stage.arc", settings: { foreground: "#dadada" } },
    { scope: "variable.other.arc", settings: { foreground: "#dadada" } },
  ],
  light: [
    { scope: "keyword.control.arc", settings: { foreground: "#CC255F" } },
    { scope: "keyword.other.arc", settings: { foreground: "#CC255F" } },
    { scope: "keyword.operator.logical.arc", settings: { foreground: "#CC255F" } },
    { scope: "keyword.operator.arithmetic.arc", settings: { foreground: "#292929" } },
    { scope: "keyword.operator.comparison.arc", settings: { foreground: "#292929" } },
    { scope: "keyword.operator.assignment.arc", settings: { foreground: "#292929" } },
    {
      scope: "keyword.operator.assignment.declare.arc",
      settings: { foreground: "#292929" },
    },
    {
      scope: "keyword.operator.assignment.stateful.arc",
      settings: { foreground: "#B45000" },
    },
    { scope: "keyword.operator.channel.arc", settings: { foreground: "#292929" } },
    { scope: "keyword.operator.transition.arc", settings: { foreground: "#BE3E4A" } },
    { scope: "keyword.operator.flow.arc", settings: { foreground: "#0097A7" } },
    { scope: "constant.language.boolean.arc", settings: { foreground: "#CC255F" } },
    { scope: "constant.language.null.arc", settings: { foreground: "#CC255F" } },
    { scope: "string.quoted.double.arc", settings: { foreground: "#0A7D00" } },
    { scope: "string.quoted.single.arc", settings: { foreground: "#0A7D00" } },
    { scope: "constant.numeric", settings: { foreground: "#0A7D00" } },
    { scope: "support.type.primitive.arc", settings: { foreground: "#267F99" } },
    { scope: "support.type.composite.arc", settings: { foreground: "#267F99" } },
    { scope: "support.type.channel.arc", settings: { foreground: "#0070C1" } },
    { scope: "comment", settings: { foreground: "#9DA5B4" } },
    { scope: "entity.name.function.arc", settings: { foreground: "#3774D0" } },
    { scope: "support.function.builtin.arc", settings: { foreground: "#3774D0" } },
    {
      scope: "support.function.builtin.stage.arc",
      settings: { foreground: "#292929" },
    },
    { scope: "entity.name.type.sequence.arc", settings: { foreground: "#292929" } },
    { scope: "entity.name.type.stage.arc", settings: { foreground: "#292929" } },
    { scope: "variable.other.arc", settings: { foreground: "#292929" } },
  ],
};

const applySemanticTokenColors = async (): Promise<destructor.Async> => {
  try {
    const vscode = await import("vscode");
    const config = vscode.workspace.getConfiguration("editor");

    // Apply semantic token colors
    const semanticColorCustomizations = {
      "[Default Dark+]": {
        rules: ARC_SEMANTIC_TOKEN_COLORS.dark,
      },
      "[Default Light+]": {
        rules: ARC_SEMANTIC_TOKEN_COLORS.light,
      },
    };
    await config.update(
      "semanticTokenColorCustomizations",
      semanticColorCustomizations,
      vscode.ConfigurationTarget.Global,
    );

    // Apply TextMate token colors (for hover popups)
    const textmateColorCustomizations = {
      "[Default Dark+]": {
        textMateRules: ARC_TEXTMATE_RULES.dark,
      },
      "[Default Light+]": {
        textMateRules: ARC_TEXTMATE_RULES.light,
      },
    };
    await config.update(
      "tokenColorCustomizations",
      textmateColorCustomizations,
      vscode.ConfigurationTarget.Global,
    );
  } catch (error) {
    console.warn("Failed to apply Arc semantic token colors:", error);
  }
  return async () => {};
};

export const EXTENSIONS: Extension[] = [];

export const SERVICES = [registerArcLanguage, applySemanticTokenColors, startArcLSP];
