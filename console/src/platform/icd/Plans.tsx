// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as PForm } from "@synnaxlabs/lyra/form";
import { ICD as PICD } from "@synnaxlabs/pluto";
import { type deep, observe } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useContext,
  useEffect,
  useState,
  useSyncExternalStore,
} from "react";

import {
  type BoundMessage,
  type MessagePlan,
  planMessage,
} from "@/platform/icd/configure";
import { useFormats } from "@/platform/icd/Formats";
import { jsonFields, readFields, type ReadSource } from "@/platform/icd/read";
import { Task } from "@/platform/task";

/** @returns what the device stores for a message that a source reads. */
export type PropsOf = (
  bound: BoundMessage<ReadSource>,
) => Task.ReadChannelProps | undefined;

interface Store {
  get: (read: string) => MessagePlan | undefined;
  set: (read: string, plan: MessagePlan | undefined) => void;
  subscribe: (onChange: () => void) => () => void;
}

const createStore = (): Store => {
  const plans = new Map<string, MessagePlan>();
  const changes = new observe.BaseObserver<void>();
  return {
    get: (read) => plans.get(read),
    set: (read, plan) => {
      if (plan == null) plans.delete(read);
      else plans.set(read, plan);
      changes.notify();
    },
    subscribe: (onChange) => changes.onChange(onChange),
  };
};

const Context = createContext<Store | null>(null);
Context.displayName = "ICD.PlansContext";

const useStore = (): Store => {
  const store = useContext(Context);
  if (store == null) throw new Error("Render ICD.Planner and ICD.usePlan in ICD.Plans");
  return store;
};

/** Holds the plan of each message that a source reads, for Planner and usePlan. */
export const Plans = ({ children }: PropsWithChildren): ReactElement => {
  const [store] = useState(createStore);
  return <Context value={store}>{children}</Context>;
};

const readKey = (path: deep.Path<ReadSource>, read: number): string =>
  String(path.messages[read]);

export interface PlannerProps {
  path: deep.Path<ReadSource>;
  /** The index of the message in the source's reads. */
  read: number;
  /** The prefix of the names of the channels that deploying creates. */
  prefix: string;
  propsOf: PropsOf;
}

/**
 * Plans the channels that deploying gives a message that a source reads, for usePlan.
 * Plans again when the source, its message, or what the device stores change.
 */
export const Planner = ({ path, read, prefix, propsOf }: PlannerProps): null => {
  const store = useStore();
  const formats = useFormats();
  const source = PForm.useFieldValue(path);
  const readMessage = source.messages[read];
  const message = PForm.useFieldValue(PICD.formPath.messages[readMessage.message], {
    optional: true,
    ctx: formats.get(source.icd),
  });
  const bound =
    message == null
      ? undefined
      : { source, read, message, fields: readFields(jsonFields(message), readMessage) };
  const props = bound == null ? undefined : propsOf(bound);
  const plan = Task.useChannelPlan({ bound, prefix, props }, async (client) =>
    bound == null ? undefined : await planMessage(client, bound, prefix, props),
  );
  const key = readKey(path, read);
  useEffect(() => {
    store.set(key, plan);
    return () => store.set(key, undefined);
  }, [key, plan]);
  return null;
};

/**
 * @returns the plan of a message that a source reads, or undefined until its Planner
 * plans it.
 */
export const usePlan = (
  path: deep.Path<ReadSource>,
  read: number,
): MessagePlan | undefined => {
  const store = useStore();
  const key = readKey(path, read);
  const get = () => store.get(key);
  return useSyncExternalStore(store.subscribe, get, get);
};
