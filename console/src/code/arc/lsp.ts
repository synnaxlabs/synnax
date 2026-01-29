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

import { grammarRaw as arcGrammarRaw } from "@synnaxlabs/arc";
import arcLanguageConfigurationRaw from "@/code/arc/language-configuration.json?raw";
import { type Extension } from "@/code/init/initialize";

export const LANGUAGE = "arc";

const TOKEN_CONFIG = {
  keyword: {
    dark: "#CC255F",
    light: "#CC255F",
    scopes: [
      "keyword.control.arc",
      "keyword.other.arc",
      "keyword.operator.logical.arc",
      "constant.language.boolean.arc",
      "constant.language.null.arc",
    ],
  },
  operator: {
    dark: "#dadada",
    light: "#292929",
    scopes: [
      "keyword.operator.arithmetic.arc",
      "keyword.operator.comparison.arc",
      "keyword.operator.assignment.arc",
      "keyword.operator.assignment.declare.arc",
      "keyword.operator.channel.arc",
    ],
  },
  statefulVariable: {
    dark: "#E5A84B",
    light: "#B45000",
    scopes: ["keyword.operator.assignment.stateful.arc"],
  },
  edgeOneShot: {
    dark: "#E06C75",
    light: "#BE3E4A",
    scopes: ["keyword.operator.transition.arc"],
  },
  edgeContinuous: {
    dark: "#56c8d8",
    light: "#0097A7",
    scopes: ["keyword.operator.flow.arc"],
  },
  string: {
    dark: "#98C379",
    light: "#0A7D00",
    scopes: ["string.quoted.double.arc", "string.quoted.single.arc"],
  },
  number: {
    dark: "#98C379",
    light: "#0A7D00",
    scopes: ["constant.numeric"],
  },
  type: {
    dark: "#4EC9B0",
    light: "#267F99",
    scopes: ["support.type.primitive.arc", "support.type.composite.arc"],
  },
  channel: {
    dark: "#61AFEF",
    light: "#0070C1",
    scopes: ["support.type.channel.arc"],
  },
  comment: {
    dark: "#5C6370",
    light: "#9DA5B4",
    scopes: ["comment"],
  },
  function: {
    dark: "#556bf8",
    light: "#3774D0",
    scopes: ["entity.name.function.arc", "support.function.builtin.arc"],
  },
  stage: {
    dark: "#dadada",
    light: "#292929",
    scopes: ["support.function.builtin.stage.arc", "entity.name.type.stage.arc"],
  },
  sequence: {
    dark: "#dadada",
    light: "#292929",
    scopes: ["entity.name.type.sequence.arc"],
  },
  variable: {
    dark: "#dadada",
    light: "#292929",
    scopes: ["variable.other.arc"],
  },
  block: {
    dark: "#dadada",
    light: "#292929",
    scopes: [],
  },
  parameter: {
    dark: "#dadada",
    light: "#292929",
    scopes: [],
  },
  config: {
    dark: "#dadada",
    light: "#292929",
    scopes: [],
  },
  input: {
    dark: "#dadada",
    light: "#292929",
    scopes: [],
  },
  output: {
    dark: "#dadada",
    light: "#292929",
    scopes: [],
  },
  constant: {
    dark: "#dadada",
    light: "#292929",
    scopes: [],
  },
  unit: {
    dark: "#dadada",
    light: "#292929",
    scopes: [],
  },
} as const;

export type SemanticTokenType = keyof typeof TOKEN_CONFIG;

type Theme = "dark" | "light";

export type SemanticTokenColors = Record<SemanticTokenType, string>;

export interface ThemedSemanticTokenColors {
  dark: SemanticTokenColors;
  light: SemanticTokenColors;
}

const deriveSemanticTokenColors = (): ThemedSemanticTokenColors => ({
  dark: Object.fromEntries(
    Object.entries(TOKEN_CONFIG).map(([key, value]) => [key, value.dark]),
  ) as SemanticTokenColors,
  light: Object.fromEntries(
    Object.entries(TOKEN_CONFIG).map(([key, value]) => [key, value.light]),
  ) as SemanticTokenColors,
});

export const SEMANTIC_TOKEN_COLORS: ThemedSemanticTokenColors =
  deriveSemanticTokenColors();

interface TextMateRule {
  scope: string;
  settings: { foreground: string };
}

const deriveTextMateRules = (theme: Theme): TextMateRule[] =>
  Object.values(TOKEN_CONFIG).flatMap((config) =>
    config.scopes.map((scope) => ({
      scope,
      settings: { foreground: config[theme] },
    })),
  );

const TEXTMATE_RULES = {
  dark: deriveTextMateRules("dark"),
  light: deriveTextMateRules("light"),
};

const NOOP_DISPOSER = () => ({ dispose: () => {} });

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
  let onMessageCallback: ((message: Message) => void) | null = null;

  const receiveLoop = async () => {
    try {
      while (!isClosed) {
        const [msg, err] = await stream.receive();
        if (err != null) {
          onErrorCallback?.(err);
          break;
        }
        if (msg == null) break;
        try {
          const parsed = JSON.parse(msg.content);
          onMessageCallback?.(parsed);
        } catch (parseError) {
          onErrorCallback?.(
            parseError instanceof Error ? parseError : new Error(String(parseError)),
          );
        }
      }
    } finally {
      isClosed = true;
      onCloseCallback?.();
    }
  };

  const reader: MessageReader = {
    listen: (callback) => {
      onMessageCallback = callback as (message: Message) => void;
      receiveLoop().catch((err) => onErrorCallback?.(err));
      return { dispose: () => (onMessageCallback = null) };
    },
    dispose: () => (isClosed = true),
    onError: (callback) => {
      onErrorCallback = callback;
      return { dispose: () => (onErrorCallback = null) };
    },
    onClose: (callback) => {
      onCloseCallback = callback;
      return { dispose: () => (onCloseCallback = null) };
    },
    onPartialMessage: NOOP_DISPOSER,
  };

  const writer: MessageWriter = {
    write: async (message) => {
      if (isClosed) throw new Error("Stream is closed");
      stream.send({ content: JSON.stringify(message) });
    },
    dispose: () => (isClosed = true),
    onError: (callback) => {
      onErrorCallback = (err: Error) => callback([err, undefined, undefined]);
      return { dispose: () => (onErrorCallback = null) };
    },
    onClose: (callback) => {
      onCloseCallback = callback;
      return { dispose: () => (onCloseCallback = null) };
    },
    end: () => {
      isClosed = true;
      stream.closeSend();
    },
  };

  return { reader, writer };
};

let synnaxClient: Synnax | null = null;
let lspDestructor: destructor.Async | null = null;

export const setSynnaxClient = async (client: Synnax | null): Promise<void> => {
  if (lspDestructor != null) {
    await lspDestructor();
    lspDestructor = null;
  }
  synnaxClient = client;
  if (client == null) return;
  lspDestructor = await startArcLSP();
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

const GRAMMAR_PATH = "./arc.tmLanguage.json";
const LANGUAGE_CONFIGURATION_PATH = "./language-configuration.json";
const GRAMMAR_DATA_URL = `data:application/json;base64,${btoa(arcGrammarRaw)}`;
const LANGUAGE_CONFIG_DATA_URL = `data:application/json;base64,${btoa(arcLanguageConfigurationRaw)}`;

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

  registerFileUrl(GRAMMAR_PATH, GRAMMAR_DATA_URL);
  registerFileUrl(LANGUAGE_CONFIGURATION_PATH, LANGUAGE_CONFIG_DATA_URL);

  return async () => {};
};

const applySemanticTokenColors = async (): Promise<destructor.Async> => {
  try {
    const vscode = await import("vscode");
    const config = vscode.workspace.getConfiguration("editor");

    await config.update(
      "semanticTokenColorCustomizations",
      {
        "[Default Dark+]": { rules: SEMANTIC_TOKEN_COLORS.dark },
        "[Default Light+]": { rules: SEMANTIC_TOKEN_COLORS.light },
      },
      vscode.ConfigurationTarget.Global,
    );

    await config.update(
      "tokenColorCustomizations",
      {
        "[Default Dark+]": { textMateRules: TEXTMATE_RULES.dark },
        "[Default Light+]": { textMateRules: TEXTMATE_RULES.light },
      },
      vscode.ConfigurationTarget.Global,
    );
  } catch (error) {
    console.warn("Failed to apply Arc semantic token colors:", error);
  }
  return async () => {};
};

export const EXTENSIONS: Extension[] = [];

export const SERVICES = [registerArcLanguage, applySemanticTokenColors];
