// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type * as monacoT from "@codingame/monaco-vscode-editor-api";
import { errors } from "@synnaxlabs/x";

import { type Language, registerLanguage } from "@/code/language";

const WORKER_LOADERS: Partial<Record<string, () => Worker>> = {
  TextEditorWorker: () =>
    new Worker(
      new URL(
        "@codingame/monaco-vscode-editor-api/esm/vs/editor/editor.worker.js",
        import.meta.url,
      ),
      { type: "module" },
    ),
  TextMateWorker: () =>
    new Worker(
      new URL(
        "@codingame/monaco-vscode-textmate-service-override/worker",
        import.meta.url,
      ),
      { type: "module" },
    ),
};

const getWorker = (_: string, label: string) => {
  const workerFactory = WORKER_LOADERS[label];
  if (workerFactory != null) return workerFactory();
  throw new Error(`Worker ${label} not found`);
};

export interface InitializeProps {
  languages: Language[];
}

export interface InitializeReturn {
  monaco: typeof monacoT;
}

let initPromise: Promise<InitializeReturn> | null = null;

/** initializeMonaco performs the one-time, ordered bootstrap of the monaco-vscode-api
 * runtime and registers the given languages. It is memoized: monaco-vscode-api can only
 * be initialized once per page, so every caller shares a single promise. */
export const initializeMonaco = (props: InitializeProps): Promise<InitializeReturn> => {
  if (initPromise != null) return initPromise;
  initPromise = doInitialize(props).catch((e: unknown) => {
    initPromise = null;
    throw errors.fromUnknown(e);
  });
  return initPromise;
};

const doInitialize = async ({
  languages,
}: InitializeProps): Promise<InitializeReturn> => {
  self.MonacoEnvironment = { getWorker };
  // Everything in this batch must load BEFORE initialize(). In particular importing
  // vscode/localExtensionHost registers the participant that publishes the default vscode
  // api during initialize(); deferring it past initialize() leaves the api permanently
  // unavailable (the LSP client and token theming both depend on it).
  const [
    ,
    { initialize },
    { default: getTextMateServiceOverride },
    { default: getThemeServiceOverride },
    { default: getLanguagesServiceOverride },
  ] = await Promise.all([
    import("@codingame/monaco-vscode-theme-defaults-default-extension"),
    import("@codingame/monaco-vscode-api"),
    import("@codingame/monaco-vscode-textmate-service-override"),
    import("@codingame/monaco-vscode-theme-service-override"),
    import("@codingame/monaco-vscode-languages-service-override"),
    import("@codingame/monaco-vscode-extension-api/localExtensionHost"),
  ]);
  await initialize({
    ...getTextMateServiceOverride(),
    ...getThemeServiceOverride(),
    ...getLanguagesServiceOverride(),
  });
  const monaco = await import("@codingame/monaco-vscode-editor-api");
  for (const language of languages) await registerLanguage(language);
  return { monaco };
};
