// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, schematic } from "@synnaxlabs/client";
import { color, type text } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useState } from "react";

import { Component } from "@/component";
import { Form as Base } from "@/form";
import { Input } from "@/input";
import { Project } from "@/project";
import { Form } from "@/schematic/node/common/form";
import { Orientation } from "@/schematic/node/common/orientation";
import { PAGE_ICONS } from "@/schematic/node/general/offPageReference/config";
import { type FormProps } from "@/schematic/node/spec";
import { Select } from "@/select";
import { Status } from "@/status/base";
import { Synnax } from "@/synnax";
import { Theming } from "@/theming";
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
      (v: string) => onChange(v === "single"),
      [onChange],
    );
    return (
      <Select.Buttons
        value={value ? "single" : "double"}
        onChange={handleChange}
        keys={CLICK_MODE_KEYS}
      >
        <Select.Button itemKey="single">Single</Select.Button>
        <Select.Button itemKey="double">Double</Select.Button>
      </Select.Buttons>
    );
  },
);

const selectKey = (page?: schematic.Page | null): string =>
  page == null || page.key.length === 0 ? "" : ontology.idToString(page);

const useHandlePageChange = (): ((v: string | null) => void) => {
  const theme = Theming.use();
  const ctx = Base.useContext();
  return useCallback(
    (v: string | null) => {
      const prev = ctx.get<schematic.Page | undefined>("page", {
        optional: true,
      })?.value;
      const cleared = v == null || v.length === 0;
      ctx.set(
        "page",
        cleared ? undefined : schematic.pageZ.parse(ontology.stringIDZ.parse(v)),
      );
      const hadPage = prev != null && prev.key.length > 0;
      if (!hadPage && !cleared) ctx.set("color", color.hex(theme.colors.primary.z));
    },
    [ctx, theme],
  );
};

export const OffPageReferenceForm = ({ schematicKey }: FormProps): ReactElement => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const [siblings, setSiblings] = useState<Select.StaticEntry<string>[]>([]);
  useEffect(() => {
    setSiblings([]);
    if (client == null || schematicKey == null) return;
    handleError(async () => {
      const children = await Project.retrieveChildren(client, {
        resourceID: schematic.ontologyID(schematicKey),
        types: [...schematic.PAGE_TYPES],
      });
      setSiblings(
        children.flatMap(({ key, name, type }) => {
          const pageType = schematic.pageTypeZ.safeParse(type);
          if (!pageType.success) return [];
          const PageIcon = PAGE_ICONS[pageType.data];
          return {
            key: ontology.idToString({ type: pageType.data, key }),
            name,
            icon: <PageIcon />,
          };
        }),
      );
    }, "Failed to retrieve project pages");
  }, [client, schematicKey, handleError]);
  const handlePageChange = useHandlePageChange();
  const page = Base.useFieldValue<schematic.Page | undefined>("page", {
    optional: true,
  });
  return (
    <Base.Sections x>
      <Base.Section title="Label">
        <Base.TextField path="label.label" label="Label" padHelpText={false} />
        <Base.Field<text.Level>
          hideIfNull
          path="label.level"
          label="Size"
          padHelpText={false}
        >
          {Form.SelectTextLevel}
        </Base.Field>
      </Base.Section>
      <Base.Section title="Navigation">
        <Input.Item label="Page" padHelpText={false}>
          <Select.Static
            value={selectKey(page)}
            onChange={handlePageChange}
            data={siblings}
            resourceName="page"
            emptyContent="No other pages in this project"
            allowNone
          />
        </Input.Item>
        <Base.Field<boolean>
          path="dblClickNavDisabled"
          label="Click mode"
          padHelpText={false}
          hideIfNull={false}
        >
          {ClickModeSelect}
        </Base.Field>
      </Base.Section>
      <Base.Section title="Appearance">
        <Form.ColorField path="color" />
      </Base.Section>
      <Orientation.Section path="" hideOuter />
    </Base.Sections>
  );
};
