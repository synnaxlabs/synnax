// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Form as Base } from "@/form";
import { Group } from "@/schematic/group";
import { Form } from "@/schematic/node/common/form";
import { REGISTRY } from "@/schematic/node/registry";
import { Theming } from "@/theming";

const theme = Theming.themeZ.parse(Theming.SYNNAX_THEMES.synnaxDark);
const NOTICE = "Style editing disabled while grouped. Ungroup to edit.";
const schema = z.record(z.string(), z.unknown());

const SPECS = Object.values(REGISTRY);

const configOf = (spec: (typeof SPECS)[number]): Record<string, unknown> =>
  spec.defaultConfig(theme);

const groupOf = (members: string[]): record.Unknown => ({
  ...(REGISTRY.groupBox.defaultConfig(theme) as record.Unknown),
  members,
});

const symbolNode = (key: string, x = 0, y = 0): schematic.Node => ({
  key,
  position: { x, y },
  zIndex: 0,
});

const apply = (
  configs: Record<string, record.Unknown>,
  actions: schematic.Action[],
): void => {
  for (const a of actions)
    if (a.type === "set_node" && a.setNode.config != null)
      configs[a.setNode.node.key] = a.setNode.config;
    else if (a.type === "remove_node") delete configs[a.removeNode.key];
};

interface HostProps extends PropsWithChildren {
  values: Record<string, unknown>;
  locked: boolean;
}

// Mounts a symbol form over the given config with the style lock set or cleared.
const Host = ({ values, locked, children }: HostProps): ReactElement => {
  const methods = Base.use({ values, schema });
  return (
    <Form.StyleLockContext value={locked}>
      <Base.Form<typeof schema> {...methods}>{children}</Base.Form>
    </Form.StyleLockContext>
  );
};

describe("Form.StyleLock", () => {
  describe("Wrapper", () => {
    it("should replace lockable fields with the notice when locked", () => {
      const { getByText, queryByText } = render(
        <Form.StyleLockContext value>
          <Form.Wrapper lockable>
            <div>style fields</div>
          </Form.Wrapper>
        </Form.StyleLockContext>,
      );
      expect(getByText(NOTICE)).toBeDefined();
      expect(queryByText("style fields")).toBeNull();
    });

    it("should render lockable fields when the lock is off", () => {
      const { getByText, queryByText } = render(
        <Form.Wrapper lockable>
          <div>style fields</div>
        </Form.Wrapper>,
      );
      expect(getByText("style fields")).toBeDefined();
      expect(queryByText(NOTICE)).toBeNull();
    });

    it("should leave unmarked fields alone when locked", () => {
      const { getByText, queryByText } = render(
        <Form.StyleLockContext value>
          <Form.Wrapper>
            <div>telemetry fields</div>
          </Form.Wrapper>
        </Form.StyleLockContext>,
      );
      expect(getByText("telemetry fields")).toBeDefined();
      expect(queryByText(NOTICE)).toBeNull();
    });
  });

  describe("symbol forms", () => {
    SPECS.forEach((spec) => {
      it(`should lock the ${spec.key} style fields while grouped`, () => {
        const { getByText } = render(
          <Host values={configOf(spec)} locked>
            <spec.Form />
          </Host>,
        );
        expect(getByText(NOTICE)).toBeDefined();
      });

      it(`should render the ${spec.key} style fields when ungrouped`, () => {
        const { queryByText } = render(
          <Host values={configOf(spec)} locked={false}>
            <spec.Form />
          </Host>,
        );
        expect(queryByText(NOTICE)).toBeNull();
      });
    });
  });

  describe("other tabs while locked", () => {
    it("should keep the light telemetry tab editable", () => {
      const spec = REGISTRY.light;
      const { getByText, queryByText } = render(
        <Host values={configOf(spec)} locked>
          <spec.Form />
        </Host>,
      );
      fireEvent.click(getByText("Telemetry"));
      expect(queryByText(NOTICE)).toBeNull();
      expect(getByText("Lower threshold")).toBeDefined();
    });

    it("should keep the setpoint control tab editable", () => {
      const spec = REGISTRY.setpoint;
      const { getByText, queryByText } = render(
        <Host values={configOf(spec)} locked>
          <spec.Form />
        </Host>,
      );
      fireEvent.click(getByText("Control"));
      expect(queryByText(NOTICE)).toBeNull();
    });

    it("should keep the select options tab editable", () => {
      const spec = REGISTRY.select;
      const { getByText, queryByText } = render(
        <Host values={configOf(spec)} locked>
          <spec.Form />
        </Host>,
      );
      fireEvent.click(getByText("Options"));
      expect(queryByText(NOTICE)).toBeNull();
    });
  });

  describe("lock cycle", () => {
    const measure = (): { width: number; height: number } => ({
      width: 10,
      height: 10,
    });
    const nodes = [symbolNode("a"), symbolNode("b", 50, 50)];
    // The lock derives from group membership the same way the Console derives it.
    const lockedFor = (configs: Record<string, record.Unknown>): boolean =>
      Group.buildParentOf(configs).has("a");

    it("should lock on group, unlock on ungroup, and relock on regroup", () => {
      const spec = REGISTRY.light;
      const values = configOf(spec);
      const configs: Record<string, record.Unknown> = {};
      const grouped = Group.createActions({
        selected: ["a", "b"],
        nodes,
        configs,
        measure,
      });
      apply(configs, grouped?.actions ?? []);
      const view = render(
        <Host values={values} locked={lockedFor(configs)}>
          <spec.Form />
        </Host>,
      );
      expect(view.getByText(NOTICE)).toBeDefined();
      expect(view.queryByText("Label")).toBeNull();

      const groupKey = grouped?.selection[0] ?? "";
      const ungrouped = Group.ungroupActions([groupKey, "a", "b"], configs);
      apply(configs, ungrouped?.actions ?? []);
      view.rerender(
        <Host values={values} locked={lockedFor(configs)}>
          <spec.Form />
        </Host>,
      );
      expect(view.queryByText(NOTICE)).toBeNull();
      expect(view.getByText("Label")).toBeDefined();

      const regrouped = Group.createActions({
        selected: ["a", "b"],
        nodes,
        configs,
        measure,
      });
      apply(configs, regrouped?.actions ?? []);
      view.rerender(
        <Host values={values} locked={lockedFor(configs)}>
          <spec.Form />
        </Host>,
      );
      expect(view.getByText(NOTICE)).toBeDefined();
      expect(view.queryByText("Label")).toBeNull();
    });

    it("should keep a nested member locked after the outer group ungroups", () => {
      const spec = REGISTRY.light;
      const configs: Record<string, record.Unknown> = {
        outer: groupOf(["inner", "b"]),
        inner: groupOf(["a"]),
      };
      const result = Group.ungroupActions(
        Group.withMembers(["outer"], configs),
        configs,
      );
      apply(configs, result?.actions ?? []);
      const { getByText } = render(
        <Host values={configOf(spec)} locked={lockedFor(configs)}>
          <spec.Form />
        </Host>,
      );
      expect(getByText(NOTICE)).toBeDefined();
    });
  });
});
