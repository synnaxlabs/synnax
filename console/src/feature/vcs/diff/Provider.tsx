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
import { type FC, type PropsWithChildren, type ReactElement, useState } from "react";

import { Panel } from "@/platform/panel";

export const TAB_TYPE = "vcs_diff";

/** One resource at two revs, for the diff tab to compare. */
export interface Target {
  /** resource is the compact resource ID, such as "schematic:6f1c...". */
  resource: string;
  type: string;
  name: string;
  before: vcs.Rev;
  after: vcs.Rev;
  /** beforeLabel names the before rev, such as "Last commit". */
  beforeLabel: string;
  /** afterLabel names the after rev, such as "Now". */
  afterLabel: string;
}

/** Props of the actions a {@link Provider} adds to the diff tab header. */
export interface ActionsProps {
  target: Target;
}

export interface Opened {
  target: Target;
  /** version counts the opens, so opening the same target again reads it again. */
  version: number;
}

interface ContextValue {
  opened: Opened | null;
  open: (target: Target) => void;
  Actions?: FC<ActionsProps>;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "Diff.Context",
  providerName: "Diff.Provider",
});

export interface ProviderProps extends PropsWithChildren {
  /** Actions renders in the diff tab header, such as a button that discards the
   * change. */
  Actions?: FC<ActionsProps>;
}

/** Holds the target of the diff tab. Mount it above every caller of {@link useOpen}
 * and above the panel that shows the tab. */
export const Provider = ({ Actions, children }: ProviderProps): ReactElement => {
  const [opened, setOpened] = useState<Opened | null>(null);
  const open = (target: Target) =>
    setOpened((prev) => ({ target, version: (prev?.version ?? 0) + 1 }));
  return <Context value={{ opened, open, Actions }}>{children}</Context>;
};

/** Returns the target the diff tab shows, or null before the first open. */
export const useOpened = (): Opened | null => useContext("Diff.useOpened").opened;

/** Returns a check for whether the diff tab shows the resource between two revs. */
export const useShows = (): ((
  resource: string,
  before: vcs.Rev,
  after: vcs.Rev,
) => boolean) => {
  const opened = useOpened();
  return (resource, before, after) =>
    opened?.target.resource === resource &&
    opened.target.before === before &&
    opened.target.after === after;
};

export const useActions = (): FC<ActionsProps> | undefined =>
  useContext("Diff.useActions").Actions;

/**
 * Returns a function that shows the target in the diff tab and focuses it. There is
 * one diff tab per panel: a new target replaces what it shows.
 */
export const useOpen = (): ((target: Target) => void) => {
  const { open } = useContext("Diff.useOpen");
  const openTab = Panel.useOpenTab();
  return (target) => {
    open(target);
    openTab({ variant: "view", type: TAB_TYPE }, { singleton: true });
  };
};
