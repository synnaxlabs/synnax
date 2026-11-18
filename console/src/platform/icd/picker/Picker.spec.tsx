// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ICD } from "@/platform/icd";
import { renderWithConsole } from "@/testutil";

describe("ICD.Picker.Footer", () => {
  it("should name each picked noun with its count, adding the counts of a noun", async () => {
    await renderWithConsole(
      <ICD.Picker.Footer
        counts={[
          ["field", 1],
          ["tag", 1],
          ["field", 2],
        ]}
        idle="field"
        onAdd={() => {}}
        onCancel={() => {}}
      />,
    );
    expect(screen.getByRole("button", { name: "Add 3 fields and 1 tag" })).toBeTruthy();
  });

  it("should name the idle noun and not add while nothing is picked", async () => {
    const onAdd = vi.fn();
    await renderWithConsole(
      <ICD.Picker.Footer
        counts={[["field", 0]]}
        idle="tag"
        onAdd={onAdd}
        onCancel={() => {}}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Add tags" }));
    expect(onAdd).not.toHaveBeenCalled();
  });

  it("should show its children and cancel through Cancel", async () => {
    const onCancel = vi.fn();
    await renderWithConsole(
      <ICD.Picker.Footer counts={[]} idle="field" onAdd={() => {}} onCancel={onCancel}>
        2 values selected
      </ICD.Picker.Footer>,
    );
    expect(screen.getByText("2 values selected")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onCancel).toHaveBeenCalledOnce();
  });
});
