// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bus, serial } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  createBusDevice,
  createBusLibrary,
  createBusTask,
  renderBusTask,
} from "@/feature/bus/testutil";
import { Serial } from "@/feature/serial";
import { clickDeploy, selectFromDropdown } from "@/platform/task/testutil";

const client = createTestClient();

const render = async (config: Record<string, unknown> = {}) => {
  const { library, status } = await createBusLibrary(client);
  const dev = await createBusDevice(
    client,
    Serial.Device.MAKE,
    serial.propertiesZ.parse({ port: "/dev/ttyUSB0" }),
  );
  const draft = await createBusTask(
    client,
    Serial.Task.READ_TYPE,
    Serial.Task.READ_SCHEMAS.config.parse({
      library: library.key,
      device: dev.key,
      messages: [
        { message: status.key, fields: status.fields.map((f) => ({ field: f.key })) },
      ],
      ...config,
    }),
  );
  const { container } = await renderBusTask(Serial.Task.Read, client, draft.key);
  await screen.findByRole("checkbox", { name: "Volts" });
  return { key: draft.key, container };
};

const enter = async (label: string, value: string) => {
  const input = await screen.findByLabelText(label);
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

const awaitFraming = async (key: string, framing: bus.Framing) =>
  await waitFor(async () => {
    const tsk = await client.tasks.retrieve({ key, schemas: Serial.Task.READ_SCHEMAS });
    expect(tsk.config.framing).toEqual(framing);
  });

describe("bus framing editor", () => {
  it("should store an escaped delimiter as its raw bytes", async () => {
    const { key } = await render();
    await screen.findByDisplayValue("\\n");
    await enter("Delimiter", "\\r\\n");
    await awaitFraming(key, { type: "delimiter", delimiter: "\r\n" });
    await enter("Delimiter", "\\x03");
    await awaitFraming(key, { type: "delimiter", delimiter: "\x03" });
  });

  it("should start a fixed framing from its defaults and keep its length", async () => {
    const { key } = await render();
    await selectFromDropdown("Delimiter", "Fixed length");
    await awaitFraming(key, { type: "fixed", length: 1 });
    await enter("Length", "12");
    await awaitFraming(key, { type: "fixed", length: 12 });
  });

  it("should only ask for the checksum byte order when a checksum is set", async () => {
    const { key } = await render({ framing: { type: "sync", sync: "AA55" } });
    await screen.findByDisplayValue("AA55");
    expect(screen.queryByText("Checksum byte order")).toBeNull();
    await selectFromDropdown("None", "CRC-32");
    await screen.findByText("Checksum byte order");
    await awaitFraming(key, {
      type: "sync",
      sync: "AA55",
      lengthOffset: 0,
      lengthSize: 1,
      byteOrder: "little_endian",
      lengthAdjustment: 0,
      checksum: "crc32",
      checksumByteOrder: "little_endian",
    });
  });

  it("should set the length size of a sync framing from its numeric options", async () => {
    const { key } = await render({ framing: { type: "sync", sync: "AA55" } });
    await selectFromDropdown("1 byte", "4 bytes");
    await waitFor(async () => {
      const tsk = await client.tasks.retrieve({
        key,
        schemas: Serial.Task.READ_SCHEMAS,
      });
      expect(tsk.config.framing).toMatchObject({ type: "sync", lengthSize: 4 });
    });
  });

  it("should drop the delimiter when switching to COBS or SLIP", async () => {
    const { key } = await render();
    await selectFromDropdown("Delimiter", "COBS");
    await awaitFraming(key, { type: "cobs" });
    expect(screen.queryByLabelText("Delimiter")).toBeNull();
    await selectFromDropdown("COBS", "SLIP");
    await awaitFraming(key, { type: "slip" });
  });

  describe("validation", () => {
    it("should show an empty delimiter on the delimiter field", async () => {
      const { container } = await render();
      await enter("Delimiter", "");
      await clickDeploy(container);
      await screen.findByText("Delimiter is required");
    });

    it("should show a sync that is not hex bytes on the sync field", async () => {
      const { container } = await render({ framing: { type: "sync", sync: "A" } });
      await clickDeploy(container);
      await screen.findByText("Sync must be hex bytes, such as AA55");
    });

    it("should show a fixed length of zero on the length field", async () => {
      const { container } = await render({ framing: { type: "fixed", length: 4 } });
      await enter("Length", "0");
      await clickDeploy(container);
      await screen.findByText("Length must be at least 1");
    });

    it("should show a poll rate of zero on the rate field", async () => {
      const { container } = await render();
      await enter("Poll rate", "0");
      await clickDeploy(container);
      await screen.findByText("Rate must be greater than 0");
    });
  });
});
