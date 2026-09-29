// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  can,
  type library,
  mil1553,
  serial,
  type task,
  tcp,
  udp,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type record, TimeSpan } from "@synnaxlabs/x";
import { screen } from "@testing-library/react";
import { type FC } from "react";
import { describe, expect, it } from "vitest";

import {
  createAvionicsLibrary,
  createBusDevice,
  createBusLibrary,
  createBusTask,
  createMessage,
  findFieldRow,
  renderBusTask,
} from "@/feature/bus/testutil";
import { CAN } from "@/feature/can";
import { MIL1553 } from "@/feature/mil1553";
import { Serial } from "@/feature/serial";
import { TCP } from "@/feature/tcp";
import { UDP } from "@/feature/udp";
import { type Task } from "@/platform/task";
import { deployAndAwaitTask } from "@/platform/task/testutil";

const client = createTestClient();

interface DeployParams<S extends task.Schemas> {
  Form: FC<Task.FormTabProps>;
  type: string;
  schemas: S;
  config: record.Unknown;
  /** A field of the config's message, shown once the form loads the message. */
  field: string;
  /** A value the form shows only once it has loaded the saved settings. */
  shown?: string;
}

const expectMessage = (
  messages: { message: library.EntryKey; fields: { field: library.EntryKey }[] }[],
  entry: library.MessageEntry,
) => {
  expect(messages.map((m) => m.message)).toEqual([entry.key]);
  expect(messages[0].fields.map((f) => f.field)).toEqual(
    entry.fields.map((f) => f.key),
  );
};

/** Opens a draft with the given config in its form, then deploys it from the form. */
const deploy = async <S extends task.Schemas>({
  Form,
  type,
  schemas,
  config,
  field,
  shown,
}: DeployParams<S>): Promise<task.Task<S>> => {
  const draft = await createBusTask(client, type, config);
  const { container } = await renderBusTask(Form, client, draft.key);
  if (shown != null) await screen.findByDisplayValue(shown);
  await findFieldRow(field);
  return await deployAndAwaitTask(client, container, draft.key, schemas);
};

describe("bus config round trip", () => {
  it("should keep a CAN read config", async () => {
    const lib = await createBusLibrary(client);
    const dev = await createBusDevice(
      client,
      CAN.Device.MAKE,
      can.propertiesZ.parse({}),
    );
    const { config } = await deploy({
      Form: CAN.Task.Read,
      type: CAN.Task.READ_TYPE,
      schemas: CAN.Task.READ_SCHEMAS,
      config: CAN.Task.READ_SCHEMAS.config.parse({
        library: lib.library.key,
        device: dev.key,
        messages: [createMessage(lib.engine)],
      }),
      field: "Rpm",
    });
    expect(config.library).toBe(lib.library.key);
    expect(config.device).toBe(dev.key);
    expectMessage(config.messages, lib.engine);
  });

  it("should keep a CAN write config", async () => {
    const lib = await createBusLibrary(client);
    const dev = await createBusDevice(
      client,
      CAN.Device.MAKE,
      can.propertiesZ.parse({}),
    );
    const { config } = await deploy({
      Form: CAN.Task.Write,
      type: CAN.Task.WRITE_TYPE,
      schemas: CAN.Task.WRITE_SCHEMAS,
      config: CAN.Task.WRITE_SCHEMAS.config.parse({
        library: lib.library.key,
        device: dev.key,
        messages: [createMessage(lib.engine)],
      }),
      field: "Rpm",
    });
    expect(config.device).toBe(dev.key);
    expectMessage(config.messages, lib.engine);
  });

  it("should keep a serial read config with sync framing and poll settings", async () => {
    const lib = await createBusLibrary(client);
    const dev = await createBusDevice(
      client,
      Serial.Device.MAKE,
      serial.propertiesZ.parse({ port: "/dev/ttyUSB0" }),
    );
    const framing = {
      type: "sync",
      sync: "AA55",
      lengthOffset: 2,
      lengthSize: 2,
      byteOrder: "big_endian",
      lengthAdjustment: -1,
      checksum: "crc16_modbus",
      checksumByteOrder: "big_endian",
    } as const;
    const { config } = await deploy({
      Form: Serial.Task.Read,
      type: Serial.Task.READ_TYPE,
      schemas: Serial.Task.READ_SCHEMAS,
      config: Serial.Task.READ_SCHEMAS.config.parse({
        library: lib.library.key,
        device: dev.key,
        messages: [createMessage(lib.status)],
        framing,
        rate: 5,
        timeout: TimeSpan.milliseconds(250),
      }),
      field: "Volts",
      shown: "AA55",
    });
    expect(config.framing).toEqual(framing);
    expect(config.rate).toBe(5);
    expect(config.timeout.milliseconds).toBe(250);
    expectMessage(config.messages, lib.status);
  });

  it("should keep a serial write config with SLIP framing", async () => {
    const lib = await createBusLibrary(client);
    const dev = await createBusDevice(
      client,
      Serial.Device.MAKE,
      serial.propertiesZ.parse({ port: "/dev/ttyUSB0" }),
    );
    const { config } = await deploy({
      Form: Serial.Task.Write,
      type: Serial.Task.WRITE_TYPE,
      schemas: Serial.Task.WRITE_SCHEMAS,
      config: Serial.Task.WRITE_SCHEMAS.config.parse({
        library: lib.library.key,
        device: dev.key,
        messages: [createMessage(lib.status)],
        framing: { type: "slip" },
      }),
      field: "Volts",
    });
    expect(config.framing).toEqual({ type: "slip" });
    expectMessage(config.messages, lib.status);
  });

  it("should keep a TCP read config with a two-byte delimiter", async () => {
    const lib = await createBusLibrary(client);
    const dev = await createBusDevice(
      client,
      TCP.Device.MAKE,
      tcp.propertiesZ.parse({ host: "localhost", port: 5025 }),
    );
    const { config } = await deploy({
      Form: TCP.Task.Read,
      type: TCP.Task.READ_TYPE,
      schemas: TCP.Task.READ_SCHEMAS,
      config: TCP.Task.READ_SCHEMAS.config.parse({
        library: lib.library.key,
        device: dev.key,
        messages: [createMessage(lib.status)],
        framing: { type: "delimiter", delimiter: "\r\n" },
        rate: 20,
        timeout: TimeSpan.milliseconds(50),
      }),
      field: "Volts",
      shown: "\\r\\n",
    });
    expect(config.framing).toEqual({ type: "delimiter", delimiter: "\r\n" });
    expect(config.rate).toBe(20);
    expect(config.timeout.milliseconds).toBe(50);
  });

  it("should keep a TCP write config with fixed framing", async () => {
    const lib = await createBusLibrary(client);
    const dev = await createBusDevice(
      client,
      TCP.Device.MAKE,
      tcp.propertiesZ.parse({ host: "localhost", port: 5025 }),
    );
    const { config } = await deploy({
      Form: TCP.Task.Write,
      type: TCP.Task.WRITE_TYPE,
      schemas: TCP.Task.WRITE_SCHEMAS,
      config: TCP.Task.WRITE_SCHEMAS.config.parse({
        library: lib.library.key,
        device: dev.key,
        messages: [createMessage(lib.status)],
        framing: { type: "fixed", length: 16 },
      }),
      field: "Volts",
      shown: "16",
    });
    expect(config.framing).toEqual({ type: "fixed", length: 16 });
  });

  it("should keep a UDP read config with poll settings", async () => {
    const lib = await createBusLibrary(client);
    const dev = await createBusDevice(
      client,
      UDP.Device.MAKE,
      udp.propertiesZ.parse({ port: 5000 }),
    );
    const { config } = await deploy({
      Form: UDP.Task.Read,
      type: UDP.Task.READ_TYPE,
      schemas: UDP.Task.READ_SCHEMAS,
      config: UDP.Task.READ_SCHEMAS.config.parse({
        library: lib.library.key,
        device: dev.key,
        messages: [createMessage(lib.status)],
        rate: 2,
        timeout: TimeSpan.seconds(2),
      }),
      field: "Volts",
      shown: "2000",
    });
    expect(config.rate).toBe(2);
    expect(config.timeout.milliseconds).toBe(2000);
    expectMessage(config.messages, lib.status);
  });

  it("should keep a UDP write config", async () => {
    const lib = await createBusLibrary(client);
    const dev = await createBusDevice(
      client,
      UDP.Device.MAKE,
      udp.propertiesZ.parse({ port: 5000 }),
    );
    const { config } = await deploy({
      Form: UDP.Task.Write,
      type: UDP.Task.WRITE_TYPE,
      schemas: UDP.Task.WRITE_SCHEMAS,
      config: UDP.Task.WRITE_SCHEMAS.config.parse({
        library: lib.library.key,
        device: dev.key,
        messages: [createMessage(lib.status)],
      }),
      field: "Volts",
    });
    expect(config.device).toBe(dev.key);
    expectMessage(config.messages, lib.status);
  });

  it("should keep a MIL-STD-1553 bus controller read config", async () => {
    const lib = await createAvionicsLibrary(client);
    const dev = await createBusDevice(
      client,
      MIL1553.Device.MAKE,
      mil1553.propertiesZ.parse({ backend: "simulated", role: "bus_controller" }),
    );
    const { config } = await deploy({
      Form: MIL1553.Task.Read,
      type: MIL1553.Task.READ_TYPE,
      schemas: MIL1553.Task.READ_SCHEMAS,
      config: MIL1553.Task.READ_SCHEMAS.config.parse({
        library: lib.library.key,
        device: dev.key,
        messages: [createMessage(lib.attitude)],
      }),
      field: "Pitch",
    });
    expect(config.device).toBe(dev.key);
    expectMessage(config.messages, lib.attitude);
  });

  it("should keep a MIL-STD-1553 remote terminal write config", async () => {
    const lib = await createAvionicsLibrary(client);
    const dev = await createBusDevice(
      client,
      MIL1553.Device.MAKE,
      mil1553.propertiesZ.parse({
        backend: "simulated",
        role: "remote_terminal",
        terminals: [5],
      }),
    );
    const { config } = await deploy({
      Form: MIL1553.Task.Write,
      type: MIL1553.Task.WRITE_TYPE,
      schemas: MIL1553.Task.WRITE_SCHEMAS,
      config: MIL1553.Task.WRITE_SCHEMAS.config.parse({
        library: lib.library.key,
        device: dev.key,
        messages: [createMessage(lib.status)],
      }),
      field: "Mode",
    });
    expect(config.device).toBe(dev.key);
    expectMessage(config.messages, lib.status);
  });
});
