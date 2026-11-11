// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type vcs } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Vcs } from "@/platform/vcs";

const change = (resource: string, review?: vcs.Review): vcs.Change => ({
  resource,
  type: resource.split(":")[0],
  name: resource,
  kind: "modified",
  path: `${resource}.json`,
  digest: "",
  review,
});

const SCHEMATIC = change("schematic:a");
const TASK = change("task:1");
const IN_REVIEW = change("arc:b", { number: 12, url: "https://example.com/12" });
const CHANGES = [SCHEMATIC, TASK, IN_REVIEW];

describe("selection", () => {
  it("should check every change except one an open pull request holds", () => {
    expect(Vcs.checkedResources(Vcs.ZERO_SELECTION, CHANGES)).toEqual([
      "schematic:a",
      "task:1",
    ]);
  });

  it("should let the user check a change in review", () => {
    const sel = Vcs.setChecked(Vcs.ZERO_SELECTION, IN_REVIEW.resource, true);
    expect(Vcs.isChecked(sel, IN_REVIEW)).toBe(true);
  });

  it("should check only the focused resources", () => {
    const sel = Vcs.focus([TASK.resource]);
    expect(Vcs.checkedResources(sel, CHANGES)).toEqual(["task:1"]);
  });

  it("should leave a change a later poll adds unchecked while focused", () => {
    const sel = Vcs.focus(["lineplot:c"]);
    const later = [...CHANGES, change("lineplot:c")];
    expect(Vcs.checkedResources(sel, later)).toEqual(["lineplot:c"]);
  });

  it("should check or uncheck every change at once", () => {
    expect(Vcs.checkedResources(Vcs.setAllChecked(CHANGES, false), CHANGES)).toEqual(
      [],
    );
    expect(Vcs.checkedResources(Vcs.setAllChecked(CHANGES, true), CHANGES)).toEqual([
      "schematic:a",
      "task:1",
      "arc:b",
    ]);
  });

  it("should check new changes again after a focused commit", () => {
    const sel = Vcs.forget(Vcs.focus([TASK.resource]), [TASK.resource]);
    expect(Vcs.checkedResources(sel, CHANGES)).toEqual(["schematic:a", "task:1"]);
  });
});

describe("changesOf", () => {
  it("should take the changes inside a project by its folder", () => {
    const inProject: vcs.Change = {
      ...change("schematic:p"),
      path: "projects/Wing/Schematic.json",
    };
    const other: vcs.Change = {
      ...change("schematic:q"),
      path: "projects/Fuselage/Schematic.json",
    };
    const got = Vcs.changesOf(
      [inProject, other, TASK],
      [{ id: { type: "project", key: "wing" }, name: "Wing" }],
    );
    expect(got.map(({ resource }) => resource)).toEqual(["schematic:p"]);
  });

  it("should match a resource by its ID", () => {
    const got = Vcs.changesOf(CHANGES, [{ id: { type: "task", key: "1" }, name: "T" }]);
    expect(got).toEqual([TASK]);
  });
});
