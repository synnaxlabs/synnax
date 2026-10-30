import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { EtherCAT } from "@/feature/ethercat";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { NI } from "@/feature/ni";
import { type Task } from "@/platform/task";
import { awaitTaskResourceTab, createSelectedPanel } from "@/platform/task/testutil";
import { createConsoleWrapper, createTestStore } from "@/testutil";

const client = createTestClient();

const configZ = z.looseObject({
  channels: z.array(z.looseObject({ key: z.string() })),
});

const HOOKS: [string, Task.UseCreate][] = [
  ["NI analog read", NI.Task.useCreateAnalogRead],
  ["NI analog write", NI.Task.useCreateAnalogWrite],
  ["NI counter read", NI.Task.useCreateCounterRead],
  ["NI digital read", NI.Task.useCreateDigitalRead],
  ["NI digital write", NI.Task.useCreateDigitalWrite],
  ["LabJack read", LabJack.Task.useCreateRead],
  ["LabJack write", LabJack.Task.useCreateWrite],
  ["Modbus read", Modbus.Task.useCreateRead],
  ["Modbus write", Modbus.Task.useCreateWrite],
  ["EtherCAT read", EtherCAT.Task.useCreateRead],
  ["EtherCAT write", EtherCAT.Task.useCreateWrite],
];

describe("first channel", () => {
  it.each(HOOKS)("%s", async (_, useCreate) => {
    const store = await createTestStore();
    const { wrapper } = await createConsoleWrapper({ client, store });
    const created = await createSelectedPanel(store, client);
    const { result } = renderHook(() => useCreate(), { wrapper });
    act(() => result.current());
    const key = await awaitTaskResourceTab(created);
    const tsk = await client.tasks.retrieve({ key });
    const { channels } = configZ.parse(tsk.config);
    console.log(_, JSON.stringify(channels));
    expect(channels).toHaveLength(1);
  });
});
