// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, observe } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useContext,
  useEffect,
  useState,
  useSyncExternalStore,
} from "react";

import { type MessagePlan } from "@/platform/icd/configure";
import { type ReadSource } from "@/platform/icd/read";

/** The plan of a message that a source reads. */
export interface PlannedRead {
  path: deep.Path<ReadSource>;
  /** The index of the message in the source's reads. */
  read: number;
  /** Undefined while the message has no plan. */
  plan?: MessagePlan;
}

const readKey = (path: deep.Path<ReadSource>, read: number): string =>
  String(path.messages[read]);

interface Store {
  get: (key: string) => MessagePlan | undefined;
  /** Holds plans in place of the plans it holds, and notifies of a changed one. */
  replace: (plans: PlannedRead[]) => void;
  subscribe: (onChange: () => void) => () => void;
}

const createStore = (): Store => {
  const plans = new Map<string, MessagePlan>();
  const changes = new observe.BaseObserver<void>();
  return {
    get: (key) => plans.get(key),
    replace: (next) => {
      let changed = false;
      const keys = new Set<string>();
      for (const { path, read, plan } of next) {
        const key = readKey(path, read);
        keys.add(key);
        const held = plans.get(key);
        // An equal plan keeps the held one, so its readers do not render again.
        if (held === plan || (held != null && plan != null && deep.equal(held, plan)))
          continue;
        if (plan == null) plans.delete(key);
        else plans.set(key, plan);
        changed = true;
      }
      for (const key of [...plans.keys()])
        if (!keys.has(key)) {
          plans.delete(key);
          changed = true;
        }
      if (changed) changes.notify();
    },
    subscribe: (onChange) => changes.onChange(onChange),
  };
};

const Context = createContext<Store | null>(null);
Context.displayName = "ICD.PlansContext";

export interface PlansProps extends PropsWithChildren {
  /** The plan of each message that the task reads, made together for the task. */
  plans: PlannedRead[];
}

/** Holds the plan of each message that a source reads, for usePlan. */
export const Plans = ({ plans, children }: PlansProps): ReactElement => {
  const [store] = useState(createStore);
  useEffect(() => store.replace(plans), [plans]);
  return <Context value={store}>{children}</Context>;
};

/** @returns the plan of a message that a source reads, or undefined with none. */
export const usePlan = (
  path: deep.Path<ReadSource>,
  read: number,
): MessagePlan | undefined => {
  const store = useContext(Context);
  if (store == null) throw new Error("Render ICD.usePlan in ICD.Plans");
  const key = readKey(path, read);
  const get = () => store.get(key);
  return useSyncExternalStore(store.subscribe, get, get);
};
