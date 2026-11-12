// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type vcs } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";

import { changeCount, type Choices, toChoices } from "@/feature/vcs/choices";
import { useLinkToast, useUndoToast } from "@/feature/vcs/undo";
import { Vcs } from "@/platform/vcs";

/** A change to discard: its resource ID and its name. */
export type DiscardTarget = Pick<vcs.Change, "resource" | "name">;

/** Returns a function that resets changes to the last commit, with an Undo toast. */
export const useDiscard = (): ((changes: DiscardTarget[]) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const refresh = Vcs.useRefresh();
  const undo = useUndoToast();
  return (changes) => {
    const single = changes.length === 1;
    const name = single ? changes[0].name : "";
    handleError(
      async () => {
        if (client == null || changes.length === 0) return;
        const inverse = await client.vcs.discard(
          changes.map(({ resource }) => resource),
        );
        await refresh();
        undo({
          name,
          message: single
            ? "Discarded changes"
            : `Discarded ${Vcs.count(changes.length, "change")}`,
          inverse,
          undone: "Restored changes",
        });
      },
      { message: "Failed to discard changes", name },
    );
  };
};

export interface CommitParams {
  message: string;
  resources: string[];
  pullRequest: boolean;
}

/**
 * Returns a function that commits and pushes the given resources, or opens a pull
 * request with them.
 */
export const useCommit = (): ((params: CommitParams) => Promise<void>) => {
  const client = Synnax.use();
  const setStatus = Vcs.useSetStatus();
  const update = Vcs.useUpdateUI();
  const addStatus = Status.useAdder();
  const linkToast = useLinkToast();
  return async ({ message, resources, pullRequest }) => {
    if (client == null) return;
    const res = await client.vcs.commit({ message, resources, pullRequest });
    update((prev) => ({
      ...prev,
      message: "",
      selection: Vcs.forget(prev.selection, resources),
    }));
    const status = await client.vcs.status();
    setStatus(status);
    const { configuration } = status;
    const what = Vcs.count(resources.length, "change");
    if (pullRequest) {
      linkToast({
        name: configuration,
        message: "Opened pull request",
        description: `${what} wait for review. They stay live on the Core.`,
        url: res.pullRequestUrl,
        label: "Open",
      });
      return;
    }
    const summary = `${res.commit.shortHash}: ${res.commit.message}`;
    if (status.waiting > 0 && status.remoteUrl.length > 0) {
      addStatus({
        variant: "warning",
        name: configuration,
        message: `Committed ${what}`,
        description: `The push failed, so the commit waits. The Core retries it on each fetch. ${status.error}`,
      });
      return;
    }
    addStatus({
      variant: "success",
      name: configuration,
      message: `Committed ${what}`,
      description: summary,
    });
  };
};

const LISTED_NAMES = 3;

/** Lists names in prose: "A", "A and B", "A, B, and C", or "A, B, C, and 2 more". */
const listNames = (names: string[]): string => {
  if (names.length <= 1) return names.join("");
  if (names.length === 2) return `${names[0]} and ${names[1]}`;
  if (names.length <= LISTED_NAMES)
    return `${names.slice(0, -1).join(", ")}, and ${names.at(-1)}`;
  const rest = names.length - LISTED_NAMES;
  return `${names.slice(0, LISTED_NAMES).join(", ")}, and ${rest} more`;
};

const APPLIED: Record<vcs.TargetKind, string> = {
  pull: "Pulled",
  restore: "Restored",
  load: "Loaded",
};

/** Returns a function that applies a plan, with an Undo toast. */
export const useApply = (): ((
  plan: vcs.Plan,
  choices: Choices,
  view: Vcs.PlanView,
) => Promise<void>) => {
  const client = Synnax.use();
  const setStatus = Vcs.useSetStatus();
  const update = Vcs.useUpdateUI();
  const undo = useUndoToast();
  return async (plan, choices, view) => {
    if (client == null) return;
    const inverse = await client.vcs.apply(plan.target, toChoices(plan, choices));
    const status = await client.vcs.status();
    setStatus(status);
    update((prev) => ({
      ...prev,
      view: { variant: "main" },
      message: view.message ?? prev.message,
    }));
    const changed = changeCount(plan, choices);
    const verb = APPLIED[plan.target.kind];
    const applied = plan.items.filter(
      ({ conflict, resource }) => !conflict || choices[resource] === "incoming",
    );
    const kept = plan.items.filter(({ resource }) => choices[resource] === "mine");
    const restarted = applied.filter(({ restarts }) => restarts);
    const updated = listNames(applied.map(({ name }) => name));
    const sentences: string[] = [];
    if (applied.length > 0 && restarted.length === applied.length)
      sentences.push(`Updated and restarted ${updated}.`);
    else if (applied.length > 0) {
      sentences.push(`Updated ${updated}.`);
      if (restarted.length > 0)
        sentences.push(`Restarted ${listNames(restarted.map(({ name }) => name))}.`);
    }
    if (kept.length > 0)
      sentences.push(
        `Kept your version of ${listNames(kept.map(({ name }) => name))}.`,
      );
    let message = `${verb} ${Vcs.count(changed, "change")}`;
    if (plan.target.kind === "load")
      message = `Loaded ${plan.target.configuration ?? status.configuration}`;
    else if (changed === 0) message = `${verb} with no changes`;
    undo({
      name: status.configuration,
      message,
      description: sentences.length > 0 ? sentences.join(" ") : undefined,
      inverse,
      undone:
        inverse.configuration === ""
          ? "Reverted changes"
          : `Loaded ${inverse.configuration} again`,
    });
  };
};
