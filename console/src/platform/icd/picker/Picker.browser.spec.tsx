// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/lyra/dialog";
import { treatment } from "@synnaxlabs/lyra/styletest";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";
import { page, server } from "vitest/browser";

import { ICD } from "@/platform/icd";
import { renderWithConsole } from "@/testutil";

describe("ICD.Picker.PaneHeader", () => {
  it("should show the hint of the frame under the title", async () => {
    await renderWithConsole(
      <Dialog.Frame variant="modal" visible onVisibleChange={() => {}}>
        <ICD.Picker.Root value="topics">
          <ICD.Picker.Frame itemKey="topics" hint="Select the values to read.">
            <ICD.Picker.PaneHeader
              title="plant/oven"
              free={2}
              picked={0}
              onPickAll={() => {}}
              onClear={() => {}}
            />
          </ICD.Picker.Frame>
        </ICD.Picker.Root>
      </Dialog.Frame>,
    );
    expect(treatment(screen.getByText("plant/oven"))).toBe("prose");
    expect(screen.getByText("Select the values to read.")).toBeTruthy();
  });
});

describe("ICD.Picker.Values", () => {
  it("should draw names in Inter and last values with tabular digits", async () => {
    await renderWithConsole(
      <ICD.Picker.Values
        noun="Field"
        groups={[
          {
            title: "plant/oven",
            values: [
              { key: "t", name: "temperature", sample: "21.5", dataType: "float64" },
            ],
          },
        ]}
        value={[]}
        onChange={() => {}}
      />,
    );
    expect(treatment(screen.getByText("plant/oven"))).toBe("prose");
    expect(treatment(screen.getByText("temperature"))).toBe("prose");
    expect(treatment(screen.getByText("21.5"))).toBe("numeric");
  });

  it.each([
    { columns: "without", width: 720, defaults: false },
    { columns: "with", width: 1280, defaults: true },
  ])(
    "should widen the Type column to show its longest lock label in full $columns a Default column",
    async ({ width: viewport, defaults }) => {
      const { width, height } = server.config.browser.viewport;
      onTestFinished(async () => await page.viewport(width, height));
      // The picker fills this width, which leaves room for the Type column to grow.
      await page.viewport(viewport, 900);
      const label = "Matches 2 messages";
      await renderWithConsole(
        <Dialog.Frame variant="modal" visible onVisibleChange={() => {}}>
          <ICD.Picker.Root value="topics">
            <ICD.Picker.Frame itemKey="topics" hint="Select the values to read.">
              <ICD.Picker.Values
                noun="Field"
                groups={[
                  {
                    title: "plant/oven",
                    values: [
                      {
                        key: "t",
                        name: "temperature",
                        sample: "21.5",
                        dataType: "float64",
                        lock: { checked: false, label },
                        default: defaults ? "" : undefined,
                      },
                    ],
                  },
                ]}
                value={[]}
                onChange={() => {}}
                onDefaultChange={defaults ? () => {} : undefined}
              />
            </ICD.Picker.Frame>
          </ICD.Picker.Root>
        </Dialog.Frame>,
      );
      const cell = screen.getByText(label);
      expect(cell.scrollWidth).toBeLessThanOrEqual(cell.clientWidth);
    },
  );

  it("should dim the name and note of each value of a dimmed group", async () => {
    await renderWithConsole(
      <Dialog.Frame variant="modal" visible onVisibleChange={() => {}}>
        <ICD.Picker.Root value="topics">
          <ICD.Picker.Values
            noun="Field"
            groups={[
              {
                title: "plant/oven",
                dimmed: true,
                values: [
                  {
                    key: "t",
                    name: "temperature",
                    sample: "21.5",
                    dataType: "float64",
                    note: "Adds to oven",
                  },
                ],
              },
            ]}
            value={[]}
            onChange={() => {}}
          />
        </ICD.Picker.Root>
      </Dialog.Frame>,
    );
    const { color } = getComputedStyle(screen.getByText("21.5"));
    expect(getComputedStyle(screen.getByText("temperature")).color).toBe(color);
    expect(getComputedStyle(screen.getByText("Adds to oven")).color).toBe(color);
  });
});

describe("ICD.Picker.PaneSkeleton", () => {
  it("should show the hint of the frame while values load", async () => {
    await renderWithConsole(
      <Dialog.Frame variant="modal" visible onVisibleChange={() => {}}>
        <ICD.Picker.Root value="topics">
          <ICD.Picker.Frame itemKey="topics" hint="Select the values to read.">
            <ICD.Picker.PaneSkeleton noun="Field" title="plant/oven" />
          </ICD.Picker.Frame>
        </ICD.Picker.Root>
      </Dialog.Frame>,
    );
    expect(treatment(screen.getByText("plant/oven"))).toBe("prose");
    expect(screen.getByText("Select the values to read.")).toBeTruthy();
  });
});

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
