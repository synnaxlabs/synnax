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
import { context } from "@synnaxlabs/lyra/context";
import { Status } from "@synnaxlabs/lyra/status";
import { errors, TimeSpan } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  use,
  useEffect,
  useRef,
  useState,
} from "react";

import { type Language, registerLanguage } from "@/code/language";
import { useLanguageServer } from "@/code/lsp";
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
  preload: () => Promise<Monaco>;
  getLanguage: (name: string) => Language | undefined;
}

// One rejected promise, not one per call: use() re-renders on an uncached promise, so a
// fresh rejection each time livelocks the boundary instead of settling it.
const NOT_MOUNTED: Promise<Monaco> = Promise.reject(
  new Error("Code.Provider is not mounted"),
);
// Standard sentinel-rejection pattern: the no-op catch marks the rejection handled so
// importing this module does not fire unhandledrejection. use() still throws it.
NOT_MOUNTED.catch(() => {});

const ZERO_CONTEXT_VALUE: ContextValue = {
  ensure: () => NOT_MOUNTED,
  preload: () => NOT_MOUNTED,
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

/** Provider makes the Monaco runtime available to descendant editors. It performs no
 * work on mount: monaco initializes on usePreload or on the first editor demand (the
 * first suspending useMonaco call). Languages with a languageServer get a reconnecting
 * client once an editor demands monaco. */
export const Provider = ({
  children,
  languages = EMPTY_LANGUAGES,
}: ProviderProps): ReactElement => {
  const [monaco, setMonaco] = useState<Monaco | null>(null);
  const initRef = useRef<Promise<Monaco> | null>(null);
  const addStatus = Status.useAdder();

  const ensure = (): Promise<Monaco> => {
    if (initRef.current == null)
      initRef.current = (async () => {
        const m = await initializeMonaco(languages);
        setMonaco(m);
        return m;
      })();
    return initRef.current;
  };

  // Leaves monaco unset, so the language servers wait for the first editor.
  const preload = (): Promise<Monaco> => initializeMonaco(languages);

  const languageMap = new Map(languages.map((l) => [l.name, l]));
  const getLanguage = (name: string) => languageMap.get(name);

  const value: ContextValue = { ensure, preload, getLanguage };

  const handleStatus = (stat: status.Status) => {
    if (stat.variant === "error" || stat.variant === "warning") addStatus(stat);
  };

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

// The longest usePreload waits for an idle moment. WebKit has no idle callbacks, so
// there it always waits this long.
const PRELOAD_DELAY = TimeSpan.seconds(2).milliseconds;

/**
 * usePreload loads the Monaco runtime once the page is idle after mount, so the first
 * editor opens without waiting for it. Language servers still connect only when an
 * editor opens. Each window that calls it loads its own runtime.
 */
export const usePreload = (): void => {
  const { preload } = useContext();
  useEffect(() => {
    // The first editor retries a failed preload and shows its error.
    const start = () => void preload().catch(() => {});
    if (typeof requestIdleCallback === "undefined") {
      const timeout = setTimeout(start, PRELOAD_DELAY);
      return () => clearTimeout(timeout);
    }
    const handle = requestIdleCallback(start, { timeout: PRELOAD_DELAY });
    return () => cancelIdleCallback(handle);
  }, [preload]);
};

/** useLanguage returns the registered Language descriptor for the given id, or undefined
 * if none is registered. */
export const useLanguage = (name: string): Language | undefined =>
  useContext().getLanguage(name);
