// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type vcs } from "@synnaxlabs/client";
import { context } from "@synnaxlabs/lyra/context";
import { Synnax } from "@synnaxlabs/pluto";
import { type destructor, errors } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useRef,
  useState,
} from "react";

import { TOOLBAR_KEY } from "@/platform/vcs/resource";
import { focus, type Selection, ZERO_SELECTION } from "@/platform/vcs/selection";
import { Session } from "@/session";

export type Tab = "changes" | "history";

/** A plan the toolbar shows for review before it applies. */
export interface PlanView {
  variant: "plan";
  target: vcs.Target;
  title: string;
  /** The rev the plan brings onto the Core, for its diffs. */
  after: vcs.Rev;
  afterLabel: string;
  /** Prefills the commit message once the plan applies. */
  message?: string;
}

/** The changes between two revs. */
export interface CompareView {
  variant: "compare";
  title: string;
  from: vcs.Rev;
  fromLabel: string;
  to: vcs.Rev;
  toLabel: string;
}

export type View = { variant: "main" } | PlanView | CompareView;

/** Limits the history to the commits that touch one resource. */
export interface HistoryFilter {
  resource: string;
  name: string;
}

export interface UIState {
  tab: Tab;
  view: View;
  selection: Selection;
  message: string;
  history: HistoryFilter | null;
  /** Counts the snapshots taken, so snapshot lists know to read again. */
  snapshots: number;
}

const ZERO_UI: UIState = {
  tab: "changes",
  view: { variant: "main" },
  selection: ZERO_SELECTION,
  message: "",
  history: null,
  snapshots: 0,
};

interface ContextValue {
  status: vcs.Status | null;
  error: string | null;
  setStatus: (status: vcs.Status) => void;
  refresh: () => Promise<void>;
  subscribe: () => destructor.Destructor;
  ui: UIState;
  update: (f: (prev: UIState) => UIState) => void;
}

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: {
    status: null,
    error: null,
    setStatus: () => {},
    refresh: async () => {},
    subscribe: () => () => {},
    ui: ZERO_UI,
    update: () => {},
  },
  displayName: "Vcs.Context",
});

const POLL_INTERVAL_MS = 1000;

const fingerprint = (status: vcs.Status): string =>
  JSON.stringify(status, (_, v: unknown) => (typeof v === "bigint" ? v.toString() : v));

/**
 * Holds the version control status of the connected Core, and the state of the version
 * control toolbar. Polls the status every second while a component that reads it is
 * mounted.
 */
export const Provider = ({ children }: PropsWithChildren): ReactElement => {
  const client = Synnax.use();
  const [status, setStatusState] = useState<vcs.Status | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [subscribers, setSubscribers] = useState(0);
  const [ui, setUI] = useState<UIState>(ZERO_UI);
  const last = useRef("");
  const inFlight = useRef(false);

  const setStatus = (next: vcs.Status) => {
    setError(null);
    const fp = fingerprint(next);
    if (fp === last.current) return;
    last.current = fp;
    setStatusState(next);
    // A draft message, such as the one a restore fills in, ends with the changes.
    if (next.changes.length === 0)
      setUI((prev) => (prev.message === "" ? prev : { ...prev, message: "" }));
  };

  const refresh = async () => {
    if (client == null) return;
    try {
      setStatus(await client.vcs.status());
    } catch (e) {
      setError(errors.fromUnknown(e).message);
    }
  };

  const poll = async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    await refresh();
    inFlight.current = false;
  };

  useEffect(() => {
    last.current = "";
    setStatusState(null);
    setError(null);
  }, [client]);

  useEffect(() => {
    if (client == null || subscribers === 0) return;
    void poll();
    const interval = setInterval(() => void poll(), POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [client, subscribers]);

  const subscribe = () => {
    setSubscribers((n) => n + 1);
    return () => setSubscribers((n) => n - 1);
  };

  return (
    <Context
      value={{ status, error, setStatus, refresh, subscribe, ui, update: setUI }}
    >
      {children}
    </Context>
  );
};

export interface UseStatusReturn {
  /** Null until the first poll answers. */
  status: vcs.Status | null;
  /** The error of the last poll, or null when it succeeded. */
  error: string | null;
}

/** Returns the version control status, and keeps it polled while mounted. */
export const useStatus = (): UseStatusReturn => {
  const { status, error, subscribe } = useContext();
  useEffect(() => subscribe(), []);
  return { status, error };
};

/**
 * Returns the last polled status without keeping the poll alive, for the many small
 * readers such as tree items. The status badge keeps the poll running.
 */
export const usePolledStatus = (): vcs.Status | null => useContext().status;

/** Returns a function that reads the status from the Core now. */
export const useRefresh = (): (() => Promise<void>) => useContext().refresh;

/** Returns a function that stores a status the Core returned from an action. */
export const useSetStatus = (): ((status: vcs.Status) => void) =>
  useContext().setStatus;

/** Returns the state of the version control toolbar. */
export const useUI = (): UIState => useContext().ui;

/** Returns a function that changes the state of the version control toolbar. */
export const useUpdateUI = (): ((f: (prev: UIState) => UIState) => void) =>
  useContext().update;

/** Returns a function that opens the version control toolbar. */
export const useOpenToolbar = (): (() => void) => {
  const dispatch = Session.useDispatch();
  return () => dispatch(Session.Nav.pinLeft({ key: TOOLBAR_KEY }));
};

const useOpenWith = (): ((f: (prev: UIState) => UIState) => void) => {
  const update = useUpdateUI();
  const open = useOpenToolbar();
  return (f) => {
    update(f);
    open();
  };
};

/** Returns a function that shows a plan in the toolbar for review. */
export const useOpenPlan = (): ((view: Omit<PlanView, "variant">) => void) => {
  const open = useOpenWith();
  return (view) =>
    open((prev) => ({ ...prev, tab: "changes", view: { variant: "plan", ...view } }));
};

/** Returns a function that shows the changes between two revs in the toolbar. */
export const useOpenCompare = (): ((view: Omit<CompareView, "variant">) => void) => {
  const open = useOpenWith();
  return (view) =>
    open((prev) => ({
      ...prev,
      tab: "changes",
      view: { variant: "compare", ...view },
    }));
};

/** Returns a function that opens the changes with only the given resources checked. */
export const useFocusCommit = (): ((resources: string[]) => void) => {
  const open = useOpenWith();
  return (resources) =>
    open((prev) => ({
      ...prev,
      tab: "changes",
      view: { variant: "main" },
      selection: focus(resources),
    }));
};

/** Returns a function that opens the history of one resource. */
export const useShowHistory = (): ((filter: HistoryFilter) => void) => {
  const open = useOpenWith();
  return (history) =>
    open((prev) => ({ ...prev, tab: "history", view: { variant: "main" }, history }));
};
