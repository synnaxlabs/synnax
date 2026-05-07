// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/charon/component";
import { CSS } from "@synnaxlabs/charon/css";
import { Flex } from "@synnaxlabs/charon/flex";
import { Select } from "@synnaxlabs/charon/select";
import type { Text } from "@synnaxlabs/charon/text";
import { Theming } from "@synnaxlabs/charon/theming";
import { schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Form as Base } from "@/form";
import { Form } from "@/schematic/node/common/form";
import { Orientation } from "@/schematic/node/common/orientation";
import { type FormProps } from "@/schematic/node/spec";
import { Workspace } from "@/workspace";
const CLICK_MODE_KEYS = ["single", "double"] as const;

const ClickModeSelect = Component.renderProp(
  ({
    value,
    onChange,
  }: {
    value: boolean;
    onChange: (v: boolean) => void;
  }): ReactElement => {
    const handleChange = useCallback(
      (v: string) => onChange(v === "double"),
      [onChange],
    );
    return (
      <Select.Buttons
        value={value ? "double" : "single"}
        onChange={handleChange}
        keys={CLICK_MODE_KEYS}
      >
        <Select.Button itemKey="single">Single</Select.Button>
        <Select.Button itemKey="double">Double</Select.Button>
      </Select.Buttons>
    );
  },
);

const useHandlePageChange = (): ((v: string) => void) => {
  const theme = Theming.use();
  const ctx = Base.useContext();
  return useCallback(
    (v: string) => {
      const prev = ctx.get<string>("page").value;
      ctx.set("page", v);
      const hadPage = prev != null && prev.length > 0;
      const hasPage = v != null && v.length > 0;
      if (!hadPage && hasPage) ctx.set("color", color.hex(theme.colors.primary.z));
    },
    [ctx, theme],
  );
};

export const OffPageReferenceForm = ({ schematicKey }: FormProps): ReactElement => {
  const { data: siblings = [] } = Workspace.useRetrieveChildren({
    resourceID: schematicKey != null ? schematic.ontologyID(schematicKey) : undefined,
    types: ["schematic"],
  });
  const handlePageChange = useHandlePageChange();
  return (
    <Form.Wrapper x align="stretch">
      <Flex.Box x grow align="stretch">
        <Base.TextField path="label.label" label="Label" padHelpText={false} grow />
        <Base.Field<string>
          path="page"
          label="Page"
          padHelpText={false}
          hideIfNull={false}
          defaultValue=""
          grow
          className={CSS.BE("symbol-form", "page-field")}
        >
          {({ value }) => (
            <Select.Static
              value={value}
              onChange={handlePageChange}
              data={siblings}
              resourceName="schematic"
              emptyContent="No other schematics in this workspace"
              allowNone
            />
          )}
        </Base.Field>
        <Base.Field<boolean>
          path="dblClickNav"
          label="Click Mode"
          padHelpText={false}
          hideIfNull={false}
          defaultValue
        >
          {ClickModeSelect}
        </Base.Field>
        <Base.Field<Text.Level>
          hideIfNull
          path="label.level"
          label="Label Size"
          padHelpText={false}
        >
          {Form.SelectTextLevel}
        </Base.Field>
        <Form.ColorField path="color" />
      </Flex.Box>
      <Orientation.Field path="" hideOuter />
    </Form.Wrapper>
  );
};
