// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";

describe("Task.Views.ItemLabel", () => {
  it.each(["JSON", "TEXT", "BINARY", "Topic", "SpB", "GET", "POST", "PUT", "PATCH"])(
    "should fit the colored kind %s in its column",
    async (kind) => {
      const { getByText } = await renderThemed(
        <Task.Views.ItemLabel kind={kind} kindColor={9}>
          plant/pump
        </Task.Views.ItemLabel>,
      );
      await document.fonts.load('600 1em "Inter Variable"');
      const el = getByText(kind, { exact: true }).element();
      expect(el.scrollWidth).toBeLessThanOrEqual(el.clientWidth);
    },
  );
});
