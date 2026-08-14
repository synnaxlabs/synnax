// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, type ontology, type Synnax } from "@synnaxlabs/client";
import { Form, Menu as PMenu } from "@synnaxlabs/pluto";
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
  Pick<device.New, "name" | "configured" | "properties" | "make" | "model">
> {}

/**
 * Creates a rack and a device on the live cluster and returns the created device.
 * Names are generated cluster-safe unless overridden.
 */
export const createTestDevice = async (
  client: Synnax,
  {
    configured = true,
    name,
    properties = {},
    make = "make",
    model = "model",
  }: CreateTestDeviceOptions = {},
): Promise<device.Device> => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  return await client.devices.create({
    key: id.create(),
    name: name ?? uniqueName("device"),
    rack: rack.key,
    location: "dev1",
    make,
    model,
    configured,
    properties,
  });
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
  /** Client backing the console wrapper; null (default) for cluster-free specs. */
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
  config: z.object({ device: z.string() }),
  rack: z.number().optional(),
});

interface DeviceFormFixtureProps extends PropsWithChildren {
  deviceKey: string;
  selectableKey?: string;
}

const DeviceFormFixture = ({
  deviceKey,
  selectableKey,
  children,
}: DeviceFormFixtureProps): ReactElement => {
  const methods = Form.use({
    schema: deviceFormSchema,
    values: { config: { device: deviceKey } },
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
  client?: Synnax | null;
}

/**
 * Renders ui inside the console provider stack and a Form context shaped the way
 * device consumers expect (config.device holding the selected device key).
 */
export const renderWithDeviceForm = async (
  ui: ReactElement,
  { deviceKey, selectableKey, client = null }: RenderWithDeviceFormOptions,
): Promise<RenderResult & { store: TestStore }> => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  const rendered = render(
    <DeviceFormFixture deviceKey={deviceKey} selectableKey={selectableKey}>
      {ui}
    </DeviceFormFixture>,
    { wrapper },
  );
  return { ...rendered, store };
};

/**
 * Shared test suite that verifies a device propertiesZ schema is tolerant of
 * missing / incomplete data — the core requirement for backward-compatible
 * device retrieval.
 *
 * @param name - human-readable integration name shown in test output.
 * @param schema - the Zod schema under test (e.g. `propertiesZ`).
 * @param zeroProperties - the ZERO_PROPERTIES constant for the integration.
 * @param partialCases - optional extra cases: each entry is `[label, input]`
 *   where `input` is a partial properties object that the schema must accept.
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
      expect(schema.safeParse(zeroProperties).success).toBe(true);
    });

    if (testEmpty)
      it("should parse completely empty properties", () => {
        expect(schema.safeParse({}).success).toBe(true);
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
        expect(schema.safeParse(input).success).toBe(true);
      });
  });
};
