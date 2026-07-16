// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type * as monacoT from "@codingame/monaco-vscode-editor-api";
import { type status } from "@synnaxlabs/client";
import { errors } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  use,
  useCallback,
  useMemo,
  useRef,
  useState,
} from "react";

import { type Language, registerLanguage } from "@/code/language";
import { useLanguageServer } from "@/code/lsp";
import { context } from "@/context";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

export type * as Monaco from "@codingame/monaco-vscode-editor-api";

type Monaco = typeof monacoT;

const WORKER_LOADERS: Partial<Record<string, () => Worker>> = {
  editorWorkerService: () =>
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

let initPromise: Promise<Monaco> | null = null;

/** initializeMonaco performs the one-time, ordered bootstrap of the monaco-vscode-api
 * runtime and registers the given languages. It is memoized: monaco-vscode-api can only
 * be initialized once per page, so every caller shares a single promise. */
const initializeMonaco = async (languages: Language[]): Promise<Monaco> => {
  initPromise ??= doInitialize(languages);
  try {
    return await initPromise;
  } catch (e) {
    initPromise = null;
    throw errors.fromUnknown(e);
  }
};

const doInitialize = async (languages: Language[]): Promise<Monaco> => {
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
  return monaco;
};

interface ContextValue {
  ensure: () => Promise<Monaco>;
  getLanguage: (name: string) => Language | undefined;
}

const ZERO_CONTEXT_VALUE: ContextValue = {
  ensure: () => Promise.reject(new Error("Code.Provider is not mounted")),
  getLanguage: () => undefined,
};

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: ZERO_CONTEXT_VALUE,
  displayName: "Code.Context",
});

const EMPTY_LANGUAGES: Language[] = [];

export interface ProviderProps extends PropsWithChildren {
  languages?: Language[];
}

/** Provider makes the Monaco runtime available to descendant editors. It performs no work
 * on mount: monaco initializes lazily on the first editor demand (the first suspending
 * useMonaco call). Languages with a languageServer get a reconnecting client once monaco
 * is ready. */
export const Provider = ({
  children,
  languages = EMPTY_LANGUAGES,
}: ProviderProps): ReactElement => {
  const [monaco, setMonaco] = useState<Monaco | null>(null);
  const initRef = useRef<Promise<Monaco> | null>(null);
  const addStatus = Status.useAdder();

  const ensure = useCallback((): Promise<Monaco> => {
    initRef.current ??= (async () => {
      const m = await initializeMonaco(languages);
      setMonaco(m);
      return m;
    })();
    return initRef.current;
  }, [languages]);

  const languageMap = useMemo(
    () => new Map(languages.map((l) => [l.name, l])),
    [languages],
  );
  const getLanguage = useCallback(
    (name: string) => languageMap.get(name),
    [languageMap],
  );

  const value = useMemo<ContextValue>(
    () => ({ ensure, getLanguage }),
    [ensure, getLanguage],
  );

  const handleStatus = useCallback(
    (stat: status.Status) => {
      if (stat.variant === "error" || stat.variant === "warning") addStatus(stat);
    },
    [addStatus],
  );

  return (
    <Context value={value}>
      {languages.map((language) =>
        language.languageServer == null ? null : (
          <LanguageServerRunner
            key={language.name}
            monaco={monaco}
            language={language}
            onStatus={handleStatus}
          />
        ),
      )}
      {children}
    </Context>
  );
};

interface LanguageServerRunnerProps {
  monaco: Monaco | null;
  language: Language;
  onStatus: (s: status.Status) => void;
}

const LanguageServerRunner = ({
  monaco,
  language,
  onStatus,
}: LanguageServerRunnerProps): null => {
  const client = Synnax.use();
  useLanguageServer({
    monaco,
    client,
    languageID: language.name,
    open: language.languageServer!,
    onStatus,
  });
  return null;
};

/** useMonaco returns the initialized Monaco runtime, suspending until it is ready and
 * triggering its lazy initialization on first use. It throws to the nearest error
 * boundary if initialization fails. */
export const useMonaco = (): Monaco => use(useContext().ensure());

/** useLanguage returns the registered Language descriptor for the given id, or undefined
 * if none is registered. */
export const useLanguage = (name: string): Language | undefined =>
  useContext().getLanguage(name);
