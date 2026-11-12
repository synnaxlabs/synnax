// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type vcs } from "@synnaxlabs/client";

/**
 * Which uncommitted changes the next commit takes. A change with no entry is checked,
 * unless an open pull request holds it or the selection is `only` its entries.
 */
export interface Selection {
  only: boolean;
  overrides: Record<string, boolean>;
}

export const ZERO_SELECTION: Selection = { only: false, overrides: {} };

type ChangeRef = Pick<vcs.Change, "resource" | "review">;

/** Returns whether the next commit takes the change. */
export const isChecked = ({ only, overrides }: Selection, change: ChangeRef): boolean =>
  overrides[change.resource] ?? (!only && change.review == null);

export const setChecked = (
  { only, overrides }: Selection,
  resource: string,
  checked: boolean,
): Selection => ({ only, overrides: { ...overrides, [resource]: checked } });

/** Checks or unchecks every change in the list. */
export const setAllChecked = (changes: ChangeRef[], checked: boolean): Selection => ({
  only: false,
  overrides: Object.fromEntries(changes.map(({ resource }) => [resource, checked])),
});

/** Checks only the given resources, including changes that appear later. */
export const focus = (resources: string[]): Selection => ({
  only: true,
  overrides: Object.fromEntries(resources.map((resource) => [resource, true])),
});

/** Returns the resources of the checked changes, in list order. */
export const checkedResources = (
  selection: Selection,
  changes: ChangeRef[],
): string[] =>
  changes.filter((c) => isChecked(selection, c)).map(({ resource }) => resource);

/**
 * Drops the entries of the given resources and checks new changes again, such as after
 * a commit takes them.
 */
export const forget = ({ overrides }: Selection, resources: string[]): Selection => {
  const next = { ...overrides };
  for (const resource of resources) delete next[resource];
  return { only: false, overrides: next };
};
