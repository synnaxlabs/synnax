// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Panel } from "@/panel";

describe("Panel.haul", () => {
  const tab: panel.Tab = {
    variant: "resource",
    key: uuid.create(),
    resource: { type: "lineplot", key: uuid.create() },
  };

  it("should round-trip a tab drag payload", () => {
    const key = uuid.create();
    const parsed = Panel.parseTabDragPayload(Panel.createTabDragPayload(key, tab));
    expect(parsed).toEqual({ panel: key, tab });
  });

  it("should return undefined for a payload that is not a tab drag", () => {
    expect(Panel.parseTabDragPayload(undefined)).toBeUndefined();
    expect(Panel.parseTabDragPayload({})).toBeUndefined();
    expect(Panel.parseTabDragPayload({ panel: "not-a-uuid", tab })).toBeUndefined();
  });
});
