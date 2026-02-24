// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";
import type * as monacoT from "monaco-editor";

const codingameImports = Promise.all([
  import("@codingame/monaco-vscode-theme-defaults-default-extension"),
  import("@codingame/monaco-vscode-api"),
  import("@codingame/monaco-vscode-textmate-service-override"),
  import("@codingame/monaco-vscode-theme-service-override"),
  import("@codingame/monaco-vscode-languages-service-override"),
]);

const WORKER_LOADERS: Partial<Record<string, () => Worker>> = {
  TextEditorWorker: () =>
    new Worker(
      new URL("monaco-editor/esm/vs/editor/editor.worker.js", import.meta.url),
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

export type Service = () => Promise<destructor.Async>;

export interface InitializeProps {
  services: Service[];
}

export interface InitializeReturn {
  monaco: typeof monacoT;
  destructor: destructor.Async;
}

let initPromise: Promise<InitializeReturn> | null = null;

export const initializeMonaco = (props: InitializeProps): Promise<InitializeReturn> => {
  if (initPromise != null) return initPromise;
  initPromise = doInitialize(props).catch((e) => {
    initPromise = null;
    throw e;
  });
  return initPromise;
};

const doInitialize = async ({
  services,
}: InitializeProps): Promise<InitializeReturn> => {
  self.MonacoEnvironment = { getWorker };
  const [
    ,
    { initialize },
    { default: getTextMateServiceOverride },
    { default: getThemeServiceOverride },
    { default: getLanguagesServiceOverride },
  ] = await codingameImports;
  await initialize({
    ...getTextMateServiceOverride(),
    ...getThemeServiceOverride(),
    ...getLanguagesServiceOverride(),
  });
  const monaco = await import("monaco-editor");
  const destructors = await Promise.all(services.map(async (s) => await s()));
  const dest: destructor.Async = async () => {
    await Promise.all(destructors.map((d) => d()));
  };
  return { monaco, destructor: dest };
};
