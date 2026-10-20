// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/arc/explorer/Explorer.css";

import { arc } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { type position } from "@synnaxlabs/lyra/position";
import { Select } from "@synnaxlabs/lyra/select";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Access, Arc } from "@synnaxlabs/pluto";
import { caseconv, location } from "@synnaxlabs/x";
import { type ReactElement, useMemo, useState } from "react";

import { Arc as PlatformArc } from "@/platform/arc";
import { CSS } from "@/platform/css";
import { type Panel } from "@/platform/panel";
import { Tree } from "@/platform/tree";
import { View } from "@/platform/view";

const startContent = <Icon.Search color={9} />;

const DIALOG_LOCATION: position.LocationPreference = {
  targetCorner: location.TOP_RIGHT,
  dialogCorner: location.TOP_LEFT,
};

const TYPE_TRIGGER_PROPS: Select.MultipleTriggerProps<arc.Type> = {
  hideTags: true,
  variant: "text",
};

const MODE_TRIGGER_PROPS: Select.MultipleTriggerProps<arc.Mode> = {
  hideTags: true,
  variant: "text",
};

const TYPE_ITEMS: ReactElement[] = arc.TYPES.map((type) => (
  <Select.Item<arc.Type> key={type} itemKey={type}>
    {PlatformArc.TYPE_ICONS[type]}
    {caseconv.capitalize(type)}
  </Select.Item>
));

const MODE_ITEMS: ReactElement[] = arc.MODES.map((mode) => (
  <Select.Item<arc.Mode> key={mode} itemKey={mode}>
    {PlatformArc.MODE_ICONS[mode]}
    {caseconv.capitalize(mode)}
  </Select.Item>
));

interface ChipsProps<K extends string> {
  label: string;
  values: K[];
  icons: Record<K, Icon.ReactElement>;
  onChange: (values: K[]) => void;
}

const Chips = <K extends string>({
  label,
  values,
  icons,
  onChange,
}: ChipsProps<K>): ReactElement | null => {
  if (values.length === 0) return null;
  return (
    <Flex.Box x pack background={0}>
      <View.FilterChip>{label}</View.FilterChip>
      {values.map((value) => (
        <Tag.Tag
          key={value}
          icon={icons[value]}
          size="small"
          onClose={() => onChange(values.filter((v) => v !== value))}
        >
          {caseconv.capitalize(value)}
        </Tag.Tag>
      ))}
    </Flex.Box>
  );
};

export const Explorer: Panel.Content = () => {
  const groupID = Arc.useGroupID({});
  const { getItem } = Arc.useList({});
  const [searchTerm, setSearchTerm] = useState("");
  const [types, setTypes] = useState<arc.Type[]>([]);
  const [modes, setModes] = useState<arc.Mode[]>([]);
  // The tree holds every name, so the term matches here. Type and mode come from the
  // list cache.
  const filter = useMemo<Tree.Filter | undefined>(() => {
    if (searchTerm === "" && types.length === 0 && modes.length === 0) return undefined;
    const term = searchTerm.toLowerCase();
    return (resource, ancestors) => {
      const a = getItem(resource.id.key);
      if (a == null) return false;
      if (types.length > 0 && !types.includes(a.type)) return false;
      if (modes.length > 0 && !modes.includes(a.mode)) return false;
      if (term === "") return true;
      return [resource, ...ancestors].some(({ name }) =>
        name.toLowerCase().includes(term),
      );
    };
  }, [searchTerm, types, modes, getItem]);
  const create = PlatformArc.useCreate();
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
  return (
    <Flex.Box y full="y" empty className={CSS.B("arc-explorer")}>
      <Flex.Box
        x
        bordered
        background={1}
        justify="between"
        align="center"
        className={CSS.BE("arc-explorer", "toolbar")}
      >
        <Dialog.Frame>
          <Dialog.Trigger
            hideCaret
            tooltip="Filter"
            tooltipLocation={location.BOTTOM_RIGHT}
          >
            <Icon.Filter />
          </Dialog.Trigger>
          <Dialog.Dialog
            background={1}
            pack={false}
            className={CSS.BE("arc-explorer", "filter-menu")}
          >
            <Menu.Menu level="small" gap="small">
              <Select.Simple<arc.Type>
                multiple
                variant="floating"
                resourceName="type"
                icon={<Icon.Arc />}
                value={types}
                onChange={setTypes}
                location={DIALOG_LOCATION}
                triggerProps={TYPE_TRIGGER_PROPS}
              >
                {TYPE_ITEMS}
              </Select.Simple>
              <Select.Simple<arc.Mode>
                multiple
                variant="floating"
                resourceName="mode"
                icon={<Icon.Text />}
                value={modes}
                onChange={setModes}
                location={DIALOG_LOCATION}
                triggerProps={MODE_TRIGGER_PROPS}
              >
                {MODE_ITEMS}
              </Select.Simple>
            </Menu.Menu>
          </Dialog.Dialog>
        </Dialog.Frame>
        <Input.Text
          size="small"
          level="h5"
          color={9}
          startContent={startContent}
          variant="text"
          value={searchTerm}
          onChange={setSearchTerm}
          placeholder="Search Arcs..."
        />
        <Chips<arc.Type>
          label="Types"
          values={types}
          icons={PlatformArc.TYPE_ICONS}
          onChange={setTypes}
        />
        <Chips<arc.Mode>
          label="Modes"
          values={modes}
          icons={PlatformArc.MODE_ICONS}
          onChange={setModes}
        />
        {hasCreatePermission && (
          <Button.Button
            onClick={create}
            tooltipLocation={location.BOTTOM_LEFT}
            tooltip="Create Arc"
            variant="filled"
          >
            <Icon.Add />
          </Button.Button>
        )}
      </Flex.Box>
      <Tree.Tree root={groupID} filter={filter} virtual={false} />
    </Flex.Box>
  );
};
