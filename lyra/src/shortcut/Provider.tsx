// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { runtime, TimeSpan } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  type RefObject,
  useEffect,
  useRef,
  useState,
} from "react";

import { context } from "@/context";
import { useSyncedRef } from "@/hooks/ref";
import {
  AreaContext,
  type AreaContextValue,
  isActive,
  type Node,
} from "@/shortcut/Area";
import { type Binding, type Chord, eventChords, format, parse } from "@/shortcut/chord";
import { isInputOrContentEditable } from "@/util/event";

/** How long a sequence waits for its next chord. */
export const SEQUENCE_TIMEOUT = TimeSpan.seconds(1);

// The chords a text field keeps for itself, though they hold Mod.
const FIELD_CHORDS = new Set([
  "Mod+A",
  "Mod+C",
  "Mod+V",
  "Mod+X",
  "Mod+Z",
  "Mod+Shift+Z",
]);

const reachesCommands = (chord: Chord): boolean =>
  chord.key === "Escape" || (chord.mod && !FIELD_CHORDS.has(format([chord])));

interface Handler {
  node: Node;
  order: number;
  run: RefObject<() => void>;
}

/** What {@link useContext} returns. Each function reads the areas when called. */
export interface ContextValue {
  /** Runs the command in the area that wins it. @returns false when no area does. */
  run: (command: string) => boolean;
  /** @returns whether an active area provides the command. */
  available: (command: string) => boolean;
  /** @returns the bindings of the command. */
  bindings: (command: string) => Binding[];
}

const [Context, useContextValue] = context.create<ContextValue>({
  displayName: "Shortcut.Context",
  providerName: "Shortcut.Provider",
});

/** @returns the commands the {@link Provider} knows and the means to run them. */
export const useContext = (): ContextValue => useContextValue("Shortcut.useContext");

const [PendingContext, usePending] = context.create<Binding>({
  defaultValue: [],
  displayName: "Shortcut.Pending",
});
/** @returns the chords of a sequence pressed so far, or none. */
export { usePending };

interface Index {
  byCommand: Map<string, Binding[]>;
  byBinding: Map<string, string[]>;
}

const index = (bindings: Record<string, string[]>): Index => {
  const byCommand = new Map<string, Binding[]>();
  const byBinding = new Map<string, string[]>();
  for (const [command, texts] of Object.entries(bindings)) {
    const list = texts.map(parse);
    byCommand.set(command, list);
    for (const binding of list) {
      const text = format(binding);
      byBinding.set(text, [...(byBinding.get(text) ?? []), command]);
    }
  }
  return { byCommand, byBinding };
};

/** Props for {@link Provider}. */
export interface ProviderProps extends PropsWithChildren {
  /** The bindings of each command, such as { "palette.open": ["Mod+P"] }. */
  bindings: Record<string, string[]>;
  /** The platform that decides what Mod means. Defaults to the one the page runs on. */
  os?: runtime.OS;
}

/**
 * Listens for keydown on the window and runs the command a press is bound to. In a
 * text field, only Escape and Mod chords reach commands, and the field keeps its own
 * Mod chords such as copy and paste. Mount one near the root.
 */
export const Provider = ({
  bindings,
  os = runtime.getOS(),
  children,
}: ProviderProps): ReactElement => {
  const parsed = index(bindings);
  const parsedRef = useSyncedRef(parsed);
  const handlers = useRef(new Map<string, Handler[]>());
  const order = useRef(0);
  const [pending, setPending] = useState<Binding>([]);
  const pendingRef = useRef<Binding>([]);
  const timeout = useRef<ReturnType<typeof setTimeout>>(undefined);

  const [area] = useState<AreaContextValue>(() => ({
    node: { parent: null, depth: 0, active: { current: true } },
    provide: (command, node, run) => {
      const handler: Handler = { node, order: order.current++, run };
      handlers.current.set(command, [
        ...(handlers.current.get(command) ?? []),
        handler,
      ]);
      return () => {
        const rest = (handlers.current.get(command) ?? []).filter((h) => h !== handler);
        handlers.current.set(command, rest);
      };
    },
  }));

  const winner = (commands: string[]): Handler | undefined => {
    let best: Handler | undefined;
    for (const command of commands)
      for (const h of handlers.current.get(command) ?? []) {
        if (!isActive(h.node)) continue;
        if (
          best == null ||
          h.node.depth > best.node.depth ||
          (h.node.depth === best.node.depth && h.order > best.order)
        )
          best = h;
      }
    return best;
  };

  const runBinding = (binding: Binding): boolean => {
    const h = winner(parsedRef.current.byBinding.get(format(binding)) ?? []);
    h?.run.current();
    return h != null;
  };

  const continues = (binding: Binding): boolean => {
    const prefix = `${format(binding)} `;
    for (const [text, commands] of parsedRef.current.byBinding)
      if (text.startsWith(prefix) && winner(commands) != null) return true;
    return false;
  };

  const wait = (binding: Binding): void => {
    clearTimeout(timeout.current);
    pendingRef.current = binding;
    setPending(binding);
    if (binding.length === 0) return;
    timeout.current = setTimeout(() => wait([]), SEQUENCE_TIMEOUT.milliseconds);
  };

  // A press runs the longest binding it completes, and waits while a longer binding
  // starts with the chords pressed so far.
  const press = (chords: Chord[]): boolean => {
    const prior = pendingRef.current;
    if (prior.length > 0) {
      const continued = chords.map((c) => [...prior, c]);
      for (const binding of continued)
        if (runBinding(binding)) {
          wait(continues(binding) ? binding : []);
          return true;
        }
      const next = continued.find(continues);
      if (next != null) {
        wait(next);
        return true;
      }
    }
    for (const chord of chords)
      if (runBinding([chord])) {
        wait(continues([chord]) ? [chord] : []);
        return true;
      }
    const started = chords.find((c) => continues([c]));
    wait(started != null ? [started] : []);
    return started != null;
  };

  const handleKeyDown = (e: KeyboardEvent): void => {
    if (e.repeat) return;
    let chords = eventChords(e, os);
    if (isInputOrContentEditable(e)) chords = chords.filter(reachesCommands);
    if (chords.length > 0 && press(chords)) e.preventDefault();
  };
  const handleKeyDownRef = useSyncedRef(handleKeyDown);

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => handleKeyDownRef.current(e);
    const onBlur = () => wait([]);
    window.addEventListener("keydown", onKeyDown);
    window.addEventListener("blur", onBlur);
    return () => {
      window.removeEventListener("keydown", onKeyDown);
      window.removeEventListener("blur", onBlur);
      clearTimeout(timeout.current);
    };
  }, []);

  const value: ContextValue = {
    run: (command) => {
      const h = winner([command]);
      h?.run.current();
      return h != null;
    },
    available: (command) => winner([command]) != null,
    bindings: (command) => parsed.byCommand.get(command) ?? [],
  };

  return (
    <Context value={value}>
      <PendingContext value={pending}>
        <AreaContext value={area}>{children}</AreaContext>
      </PendingContext>
    </Context>
  );
};
