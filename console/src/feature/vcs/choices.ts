// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type vcs } from "@synnaxlabs/client";

/** The choice made on each conflicting resource of a plan, keyed by resource. */
export type Choices = Record<string, vcs.ChoiceValue>;

const conflicts = (plan: vcs.Plan): vcs.PlanItem[] =>
  plan.items.filter(({ conflict }) => conflict);

/** Returns whether every conflict in the plan has a choice. */
export const isReady = (plan: vcs.Plan, choices: Choices): boolean =>
  conflicts(plan).every(({ resource }) => choices[resource] != null);

/** Makes the same choice on every conflict in the plan. */
export const chooseAll = (plan: vcs.Plan, choice: vcs.ChoiceValue): Choices =>
  Object.fromEntries(conflicts(plan).map(({ resource }) => [resource, choice]));

/** Returns the choices of the plan's conflicts in the form apply takes. */
export const toChoices = (plan: vcs.Plan, choices: Choices): vcs.Choice[] =>
  conflicts(plan).flatMap(({ resource }) => {
    const choice = choices[resource];
    return choice == null ? [] : [{ resource, choice }];
  });

/** Returns how many resources the apply changes on the Core. */
export const changeCount = (plan: vcs.Plan, choices: Choices): number =>
  plan.items.filter(
    ({ conflict, resource }) => !conflict || choices[resource] === "incoming",
  ).length;

export interface Group {
  kind: vcs.PlanItemKind;
  title: string;
  items: vcs.PlanItem[];
}

const GROUP_TITLES: Record<vcs.PlanItemKind, string> = {
  create: "New",
  change: "Changed",
  delete: "Deleted",
};

const KINDS: vcs.PlanItemKind[] = ["create", "change", "delete"];

/** Splits the plan into create, change, and delete groups, leaving out empty ones. */
export const group = (plan: vcs.Plan): Group[] =>
  KINDS.map((kind) => ({
    kind,
    title: GROUP_TITLES[kind],
    items: plan.items
      .filter((item) => item.kind === kind)
      .sort((a, b) => a.name.localeCompare(b.name)),
  })).filter(({ items }) => items.length > 0);

/** Returns the note a plan item shows for a running task or Arc, or null. */
export const runningNote = ({ running, restarts }: vcs.PlanItem): string | null => {
  if (!running) return null;
  return restarts ? "Restarts on apply" : "Takes effect on next start";
};
