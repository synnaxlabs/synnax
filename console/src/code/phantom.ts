// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useAsyncEffect } from "@synnaxlabs/pluto";
import type * as monaco from "monaco-editor";
import { useCallback, useRef } from "react";

import { useMonaco } from "@/code/Provider";

export interface Variable {
  key: string;
  name: string;
  value: string;
  docs?: string;
}

export interface UsePhantomReturn {
  set: ((key: string, name: string, value: string, docs?: string) => void) &
    ((variable: Variable) => void) &
    ((variables: Variable[]) => void);
  del: (key: string) => void;
}

export interface UsePhantomProps {
  language: string;
  stringifyVar: (name: string, value: string, docs?: string) => string;
  initialVars?: Variable[];
}

export const usePhantom = ({
  language,
  stringifyVar,
  initialVars,
}: UsePhantomProps): UsePhantomReturn => {
  const varsRef = useRef<Map<string, Variable>>(
    new Map(initialVars?.map((v) => [v.key, v])),
  );
  const modelRef = useRef<monaco.editor.ITextModel | null>(null);
  const monaco = useMonaco();
  const syncVars = useCallback(() => {
    const vars = Array.from(varsRef.current.entries())
      .map(([_, { name, value, docs }]) => stringifyVar(name, value, docs))
      .join("\n");
    modelRef.current?.setValue(vars);
  }, []);

  useAsyncEffect(async () => {
    if (modelRef.current != null || monaco == null) return;
    modelRef.current = monaco.editor.createModel("", language);
    syncVars();
    return () => modelRef.current?.dispose();
  }, [monaco]);

  const set = useCallback(
    (...args: any[]) => {
      if (args.length === 1 && !Array.isArray(args[0])) {
        const variable = args[0] as Variable;
        varsRef.current.set(variable.key, variable);
      } else if (args.length === 1 && Array.isArray(args[0])) {
        const variables = args[0] as Variable[];
        variables.forEach((variable) => varsRef.current.set(variable.key, variable));
      } else if (args.length >= 3) {
        const [key, name, value, docs] = args;
        varsRef.current.set(key, { key, name, value, docs });
      }
      syncVars();
    },
    [syncVars],
  );

  const del = useCallback(
    (key: string) => {
      varsRef.current.delete(key);
      syncVars();
    },
    [syncVars],
  );

  return { set, del };
};
