// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Component } from "@/component";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Form } from "@/form";
import {
  ColorControl,
  FormWrapper,
  OrientationControl,
  SelectTextLevel,
  type SymbolFormProps,
} from "@/schematic/node/common/forms";
import { Select } from "@/select";
import { type Text } from "@/text";
import { Theming } from "@/theming";
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
  const ctx = Form.useContext();
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

export const OffPageReferenceForm = ({
  schematicKey,
}: SymbolFormProps): ReactElement => {
  const { data: siblings = [] } = Workspace.useRetrieveChildren({
    resourceID: schematicKey != null ? schematic.ontologyID(schematicKey) : undefined,
    types: ["schematic"],
  });
  const handlePageChange = useHandlePageChange();
  return (
    <FormWrapper x align="stretch">
      <Flex.Box x grow align="stretch">
        <Form.TextField path="label.label" label="Label" padHelpText={false} grow />
        <Form.Field<string>
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
        </Form.Field>
        <Form.Field<boolean>
          path="dblClickNav"
          label="Click Mode"
          padHelpText={false}
          hideIfNull={false}
          defaultValue
        >
          {ClickModeSelect}
        </Form.Field>
        <Form.Field<Text.Level>
          hideIfNull
          path="label.level"
          label="Label Size"
          padHelpText={false}
        >
          {SelectTextLevel}
        </Form.Field>
        <ColorControl path="color" />
      </Flex.Box>
      <OrientationControl path="" hideOuter />
    </FormWrapper>
  );
};
