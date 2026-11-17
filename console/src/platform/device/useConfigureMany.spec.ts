// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Device } from "@/platform/device";
import { createTestDevice } from "@/platform/device/testutil";
import { createAsyncSynnaxWrapper } from "@/testutil";

const client = createTestClient();

/** @returns a free-looking identifier of the length, made of lowercase letters. */
const createIdentifier = (length: number): string =>
  Array.from({ length }, () =>
    String.fromCharCode(97 + Math.floor(Math.random() * 26)),
  ).join("");

describe("useConfigureMany", () => {
  let wrapper: FC<PropsWithChildren>;

  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const configure = async (keys: string[], ideas: string[]) => {
    const { result } = renderHook(() => Device.useConfigureMany(), { wrapper });
    await act(async () => {
      await result.current.updateAsync({
        keys,
        zeroProperties: { rate: 10 },
        suggestIdentifiers: () => ideas,
      });
    });
    return result.current;
  };

  it("should give devices with the same ideas different identifiers", async () => {
    const idea = createIdentifier(8);
    const devices = [
      await createTestDevice(client, { configured: false }),
      await createTestDevice(client, { configured: false }),
    ];
    const keys = devices.map((d) => d.key);
    expect((await configure(keys, [idea])).variant).toBe("success");
    const saved = await client.devices.retrieve({ key: keys });
    expect(saved.map((d) => d.configured)).toEqual([true, true]);
    expect(saved.map((d) => d.properties.rate)).toEqual([10, 10]);
    expect(saved.map((d) => d.properties.identifier)).toEqual([idea, `${idea}_2`]);
  });

  it("should fail when every numbered idea is too long or taken", async () => {
    const idea = createIdentifier(12);
    await createTestDevice(client, { properties: { identifier: idea } });
    const dev = await createTestDevice(client, { name: "Zq", configured: false });
    const { status } = await configure([dev.key], [idea]);
    expect(status.description).toBe(
      "Synnax cannot suggest a free identifier for Zq. Configure the device by hand.",
    );
    expect((await client.devices.retrieve({ key: dev.key })).configured).toBe(false);
  });

  it("should leave a configured device as it is", async () => {
    const kept = createIdentifier(8);
    const dev = await createTestDevice(client, { properties: { identifier: kept } });
    await configure([dev.key], [createIdentifier(8)]);
    const saved = await client.devices.retrieve({ key: dev.key });
    expect(saved.properties.identifier).toBe(kept);
    expect(saved.properties.rate).toBeUndefined();
  });
});
