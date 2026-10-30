// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, type ontology, rack, status, type Synnax } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Menu as PMenu } from "@synnaxlabs/lyra/menu";
import { id } from "@synnaxlabs/x";
import { render, type RenderResult } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Modals } from "@/platform/modals";
import { createResource } from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, type TestStore, uniqueName } from "@/testutil";

export interface CreateTestDeviceOptions extends Partial<
  Pick<device.New, "name" | "configured" | "properties" | "make" | "model" | "rack">
> {}

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
  await client.statuses.set(
    status.create<typeof rack.statusDetailsZ>({
      key: rack.statusKey(r.key),
      name: r.name,
      variant: "success",
      message: "Driver is running",
      details: { rack: r.key },
    }),
  );
  return r;
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
  }: CreateTestDeviceOptions = {},
  schemas?: device.DeviceSchemas,
): Promise<device.Device> => {
  rackKey ??= (await client.racks.create({ name: uniqueName("rack") })).key;
  return await client.devices.create(
    {
      key: id.create(),
      name: name ?? uniqueName("device"),
      rack: rackKey,
      location: "dev1",
      make,
      model,
      configured,
      properties,
    },
    schemas,
  );
};

/**
 * Builds the ontology resource the tree hands to device context-menu items, carrying
 * the configured flag in its data payload.
 */
export const createDeviceResource = (
  dev: Pick<device.Device, "key" | "name"> & { configured?: boolean },
): ontology.Resource =>
  createResource(device.ontologyID(dev.key), dev.name, {
    configured: dev.configured ?? false,
  });

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
  const rendered = render(
    <>
      <PMenu.Menu>{node}</PMenu.Menu>
      <Modals.Stack />
      <ModalStoreProbe />
    </>,
    { wrapper },
  );
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
}

const DeviceFormFixture = ({
  deviceKey,
  selectableKey,
  channels = [],
  children,
}: DeviceFormFixtureProps): ReactElement => {
  const methods = Form.use({
    schema: deviceFormSchema,
    values: { config: { device: deviceKey, channels } },
  });
  return (
    <Form.Form<typeof deviceFormSchema> {...methods}>
      {children}
      {selectableKey != null && (
        <button onClick={() => methods.set("config.device", selectableKey)}>
          select device
        </button>
      )}
    </Form.Form>
  );
};
DeviceFormFixture.displayName = "DeviceFormFixture";

export interface RenderWithDeviceFormOptions {
  deviceKey: string;
  /**
   * When set, the fixture renders a "select device" button that writes this key into
   * the form's config.device field, letting a spec flip the selection mid-test.
   */
  selectableKey?: string;
  /** The form's config.channels. Defaults to none. */
  channels?: unknown[];
  client?: Synnax | null;
}

/**
 * Renders ui inside the console provider stack and a Form context shaped like a task
 * config: config.device holds the selected device key, and config.channels holds the
 * channels.
 */
export const renderWithDeviceForm = async (
  ui: ReactElement,
  { deviceKey, selectableKey, channels, client = null }: RenderWithDeviceFormOptions,
): Promise<RenderResult & { store: TestStore }> => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  const rendered = render(
    <DeviceFormFixture
      deviceKey={deviceKey}
      selectableKey={selectableKey}
      channels={channels}
    >
      {ui}
    </DeviceFormFixture>,
    { wrapper },
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
