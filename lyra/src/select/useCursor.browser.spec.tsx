// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import { describe, expect, it } from "vitest";

import { List } from "@/list";
import { Select } from "@/select";
import { renderThemed } from "@/styletest";

const DIRECTORY =
  "/Users/synnax/Documents/projects/telemetry/console/src/platform/files/browser/data";
const FILES = ["alpha.ts", "bravo.ts", "charlie.ts"].map(
  (name) => `${DIRECTORY}/${name}`,
);

// Lists the files under the typed path.
const Files = (): ReactElement => {
  const [path, setPath] = useState("");
  const [file, setFile] = useState<string | null>(FILES[0]);
  return (
    <>
      <input aria-label="Path" value={path} onChange={(e) => setPath(e.target.value)} />
      <Select.Frame<string, record.Keyed<string>>
        data={FILES.filter((f) => f.startsWith(path))}
        value={file ?? undefined}
        onChange={setFile}
        allowNone
        initialHover={0}
      >
        <List.Scroll>
          <Select.Items<string>>
            {({ key, ...rest }) => (
              <Select.Item key={key} {...rest}>
                {rest.itemKey}
              </Select.Item>
            )}
          </Select.Items>
        </List.Scroll>
      </Select.Frame>
    </>
  );
};

// Types as an automated typist does: only microtasks pass between two keys, so React's
// scheduler never runs between them. The prototype's setter skips the value React
// tracks, so React reads each key as a change.
const typeFast = async (input: Element, text: string): Promise<void> => {
  for (let i = 1; i <= text.length; i++) {
    Reflect.set(HTMLInputElement.prototype, "value", text.slice(0, i), input);
    input.dispatchEvent(new Event("input", { bubbles: true }));
    await Promise.resolve();
  }
};

describe("Select.useCursor", () => {
  it("should keep up with fast typing that filters a hover's options", async () => {
    const { getByLabelText } = await renderThemed(<Files />);
    const input = getByLabelText("Path");
    await typeFast(input.element(), DIRECTORY);
    await expect.element(input).toHaveValue(DIRECTORY);
  });
});
