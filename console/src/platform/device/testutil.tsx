// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, rack, status, type Synnax, task } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Menu as PMenu } from "@synnaxlabs/lyra/menu";
import { Errors } from "@synnaxlabs/pluto";
import { deep, id, TimeSpan } from "@synnaxlabs/x";
import {
  act,
  render,
  type RenderOptions,
  type RenderResult,
} from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, onTestFinished } from "vitest";
import { z } from "zod";

import { type Hardware } from "@/platform/device/integration";
import { Modals } from "@/platform/modals";
import { createTaskStatus } from "@/platform/task/testutil";
import { type Tree } from "@/platform/tree";
import { createResource } from "@/platform/tree/testutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  renderSuspended,
  type TestStore,
  uniqueName,
} from "@/testutil";

const path = deep.path<z.infer<typeof deviceFormSchema>>();

export interface CreateTestDeviceOptions extends Partial<
  Pick<
    device.New,
    "name" | "configured" | "properties" | "make" | "model" | "rack" | "location"
  >
> {}

const reportRackRunning = async (client: Synnax, r: rack.Rack): Promise<void> => {
  await client.statuses.set(
    status.create<typeof rack.statusDetailsZ>({
      owner: rack.resourceID(r.key),
      name: r.name,
      variant: "success",
      message: "Driver is running",
      details: { running: true },
    }),
  );
};

/**
 * Creates a rack on the live Core whose Driver reports as running with the integration.
 * The Core marks it not running after one health check interval (5 s) without a new
 * status.
 */
export const createRunningRack = async (
  client: Synnax,
  integration: string,
): Promise<rack.Rack> => {
  const r = await client.racks.create({
    name: uniqueName("rack"),
    integrations: [integration],
  });
  await reportRackRunning(client, r);
  return r;
};

export interface CreateWarningScannerOptions {
  /** Why the scan fails. */
  description?: string;
  /** The data that the warning carries. */
  data?: unknown;
}

/** Creates a scanner on a running rack of the hardware, with a warning status. */
export const createWarningScanner = async (
  client: Synnax,
  hardware: Hardware,
  {
    description = "Cannot open en1 for raw Ethernet access.",
    data,
  }: CreateWarningScannerOptions = {},
): Promise<task.Task> => {
  const r = await createRunningRack(client, hardware.key);
  const scanner = await r.createTask({
    name: uniqueName("scanner"),
    type: hardware.scanType,
    config: {},
    internal: true,
  });
  await client.statuses.set(
    createTaskStatus({
      owner: task.resourceID(scanner.key),
      variant: "warning",
      message: "Failed to scan for devices",
      description,
      details: { running: true, rack: r.key, data },
    }),
  );
  return scanner;
};

const HEARTBEAT_PERIOD = TimeSpan.seconds(1);

/** Reports the rack as running at the rate of a Driver until the test ends. */
export const startRackHeartbeat = async (
  client: Synnax,
  r: rack.Rack,
): Promise<void> => {
  await reportRackRunning(client, r);
  const interval = setInterval(
    () => void reportRackRunning(client, r),
    HEARTBEAT_PERIOD.milliseconds,
  );
  onTestFinished(() => clearInterval(interval));
};

/**
 * Creates a device on the live Core and returns it. Without a rack, it creates one for
 * the device. Names are generated Core-safe unless overridden. Pass the schemas of the
 * make when its properties hold keys that must keep their case.
 */
export const createTestDevice = async (
  client: Synnax,
  {
    configured = true,
    name,
    properties = {},
    make = "make",
    model = "model",
    rack: rackKey,
    location = "dev1",
  }: CreateTestDeviceOptions = {},
  schemas?: device.DeviceSchemas,
): Promise<device.Device> => {
  rackKey ??= (await client.racks.create({ name: uniqueName("rack") })).key;
  return await client.devices.create(
    {
      key: id.create(),
      name: name ?? uniqueName("device"),
      rack: rackKey,
      location,
      make,
      model,
      configured,
      properties,
    },
    schemas,
  );
};

export const TEST_DEVICE_STATUS_MESSAGE = "Device reported";

/** Writes the status a Driver reports for a device. */
export const setTestDeviceStatus = async (
  client: Synnax,
  { key, name, rack }: Pick<device.Device, "key" | "name" | "rack">,
): Promise<void> => {
  await client.statuses.set(
    status.create<typeof device.statusDetailsZ>({
      owner: device.resourceID(key),
      name,
      variant: "success",
      message: TEST_DEVICE_STATUS_MESSAGE,
      details: { rack, connected: true },
    }),
  );
};

/** Builds the tree resource that the tree hands to device context-menu items. */
export const createDeviceResource = (
  dev: Pick<device.Device, "key" | "name">,
): Tree.Resource => createResource(device.resourceID(dev.key), dev.name);

export interface RenderMenuItemOptions {
  /** Client backing the console wrapper; null (default) for Core-free specs. */
  client?: Synnax | null;
}

export interface MenuItemHandle extends RenderResult {
  store: TestStore;
  /** The console wrapper backing the render, for seeding panel docs. */
  wrapper: FC<PropsWithChildren>;
  /** The window's modal store, for driving prompt dismissal from the spec. */
  modals: Session.Modals.Store;
}

/**
 * Renders a device context-menu item inside the full console provider stack, a pluto
 * menu, and a mounted modal stack, so items can open prompts and place layouts.
 */
export const renderMenuItem = async (
  node: ReactElement,
  { client = null }: RenderMenuItemOptions = {},
): Promise<MenuItemHandle> => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  const captured: { modals?: Session.Modals.Store } = {};
  const ModalStoreProbe = (): null => {
    captured.modals = Session.Modals.useStore("renderMenuItem");
    return null;
  };
  ModalStoreProbe.displayName = "ModalStoreProbe";
  let rendered!: RenderResult;
  // An item can suspend on a device fetch, and a tree bounds its menus the same way.
  await act(async () => {
    rendered = render(
      <>
        <Errors.SuspenseBoundary loading={null}>
          <PMenu.Menu>{node}</PMenu.Menu>
        </Errors.SuspenseBoundary>
        <Modals.Stack />
        <ModalStoreProbe />
      </>,
      { wrapper },
    );
  });
  if (captured.modals == null) throw new Error("modal store was not captured");
  return { ...rendered, store, wrapper, modals: captured.modals };
};

const deviceFormSchema = z.object({
  config: z.object({ device: z.string(), channels: z.array(z.unknown()) }),
  rack: z.number().optional(),
});

interface DeviceFormFixtureProps extends PropsWithChildren {
  deviceKey: string;
  selectableKey?: string;
  channels?: unknown[];
  mode?: Form.Mode;
}

const DeviceFormFixture = ({
  deviceKey,
  selectableKey,
  channels = [],
  mode,
  children,
}: DeviceFormFixtureProps): ReactElement => {
  const methods = Form.use({
    schema: deviceFormSchema,
    values: { config: { device: deviceKey, channels } },
    mode,
  });
  return (
    <Form.Form<typeof deviceFormSchema> {...methods}>
      {children}
      {selectableKey != null && (
        <button onClick={() => methods.set(path.config.device, selectableKey)}>
          select device
        </button>
      )}
    </Form.Form>
  );
};
DeviceFormFixture.displayName = "DeviceFormFixture";

export interface RenderWithDeviceFormOptions extends Pick<
  RenderOptions,
  "onCaughtError"
> {
  deviceKey: string;
  /**
   * When set, the fixture renders a "select device" button that writes this key into
   * the form's config.device field, letting a spec flip the selection mid-test.
   */
  selectableKey?: string;
  /** The form's config.channels. Defaults to none. */
  channels?: unknown[];
  /** Renders the form read-only with "preview". Defaults to "normal". */
  mode?: Form.Mode;
  client?: Synnax | null;
}

/**
 * Renders ui inside the console provider stack and a Form context shaped like a task
 * config: config.device holds the selected device key, and config.channels holds the
 * channels.
 */
export const renderWithDeviceForm = async (
  ui: ReactElement,
  {
    deviceKey,
    selectableKey,
    channels,
    mode,
    client = null,
    onCaughtError,
  }: RenderWithDeviceFormOptions,
): Promise<RenderResult & { store: TestStore }> => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  const rendered = await renderSuspended(
    <DeviceFormFixture
      deviceKey={deviceKey}
      selectableKey={selectableKey}
      channels={channels}
      mode={mode}
    >
      {ui}
    </DeviceFormFixture>,
    { wrapper, onCaughtError },
  );
  return { ...rendered, store };
};

/**
 * Shared test suite that verifies a device propertiesZ schema is tolerant of missing /
 * incomplete data — the core requirement for backward-compatible device retrieval.
 * @param name - human-readable integration name shown in test output.
 * @param schema - the Zod schema under test (e.g. `propertiesZ`).
 * @param zeroProperties - the ZERO_PROPERTIES constant for the integration.
 * @param partialCases - optional extra cases: each entry is `[label, input]` where
 * `input` is a partial properties object that the schema must accept.
 */
interface TestPropertiesSchemaOptions {
  /** Set to false to skip the empty `{}` parse test (e.g., for versioned schemas). */
  testEmpty?: boolean;
}

export const testPropertiesSchema = (
  name: string,
  schema: z.ZodType,
  zeroProperties: unknown,
  partialCases: Array<[string, unknown]> = [],
  options: TestPropertiesSchemaOptions = {},
): void => {
  const { testEmpty = true } = options;
  describe(`${name} propertiesZ`, () => {
    it("should parse ZERO_PROPERTIES", () => {
      expect(schema.validate(zeroProperties)).toBe(true);
    });

    if (testEmpty)
      it("should parse completely empty properties", () => {
        expect(schema.validate({})).toBe(true);
      });

    if (testEmpty)
      it("should produce independent objects for successive parses of empty input", () => {
        const a = schema.parse({});
        const b = schema.parse({});
        expect(a).toEqual(b);
        expect(a).not.toBe(b);
        // Mutate a nested object/array in `a` and verify `b` is unaffected.
        const aAny = a as Record<string, unknown>;
        for (const key of Object.keys(aAny)) {
          const val = aAny[key];
          if (val != null && typeof val === "object" && !Array.isArray(val)) {
            (val as Record<string, unknown>).__test = true;
            expect((b as Record<string, unknown>)[key]).not.toHaveProperty("__test");
            delete (val as Record<string, unknown>).__test;
            break;
          }
          if (Array.isArray(val)) {
            val.push("__test");
            expect((b as Record<string, unknown>)[key]).not.toContain("__test");
            val.pop();
            break;
          }
        }
      });

    for (const [label, input] of partialCases)
      it(`should parse ${label}`, () => {
        expect(schema.validate(input)).toBe(true);
      });
  });
};
