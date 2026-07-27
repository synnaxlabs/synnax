// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";
import { describe, expect, it } from "vitest";

import { ONTOLOGY_SERVICE } from "@/access/role/ontology";

const resolveIcon = (name: string): Icon.ReactElement => {
  const { icon } = ONTOLOGY_SERVICE;
  if (typeof icon !== "function") throw new Error("expected icon to be a function");
  return icon({ name } as ontology.Resource);
};

describe("role ontology icon", () => {
  it.each([
    ["Owner", Icon.Settings],
    ["Engineer", Icon.Safety],
    ["Operator", Icon.Control],
    ["Viewer", Icon.Visible],
    ["Host", Icon.TerminalOutline],
  ])("should resolve the %s icon", (name, component) => {
    expect(resolveIcon(name).type).toBe(component);
  });

  it("should fall back to the generic role icon for custom roles", () => {
    expect(resolveIcon("Custom Role").type).toBe(Icon.Role);
  });
});
