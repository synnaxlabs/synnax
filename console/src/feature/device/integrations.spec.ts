// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it, vi } from "vitest";

const flags = vi.hoisted((): { bus: boolean } => ({ bus: false }));

vi.mock("@/flags", async (importOriginal) => {
  const { FLAGS } = await importOriginal<{ FLAGS: object }>();
  return {
    FLAGS: {
      ...FLAGS,
      get bus() {
        return flags.bus;
      },
    },
  };
});

const BUS_MAKES = ["serial", "tcp"];

const BUS_COMMANDS = ["Connect serial device", "Connect TCP device"];

// The lists are built when the module loads, so each case loads it again.
const load = async () => {
  vi.resetModules();
  const { Device } = await import("@/feature/device");
  return {
    makes: Device.INTEGRATIONS.map(({ make }) => make),
    commands: Device.COMMANDS.map(({ commandName }) => commandName),
  };
};

describe("bus integrations", () => {
  beforeEach(() => {
    flags.bus = false;
  });

  it("should leave out the bus integrations and commands while the bus flag is off", async () => {
    const { makes, commands } = await load();
    for (const make of BUS_MAKES) expect(makes).not.toContain(make);
    for (const name of BUS_COMMANDS) expect(commands).not.toContain(name);
  });

  it("should list the bus integrations and commands while the bus flag is on", async () => {
    flags.bus = true;
    const { makes, commands } = await load();
    expect(makes).toEqual(expect.arrayContaining(BUS_MAKES));
    expect(commands).toEqual(expect.arrayContaining(BUS_COMMANDS));
  });
});
