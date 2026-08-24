// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/theme/Modal.css";

import { type Dispatch } from "@reduxjs/toolkit";
import { Component, Icon, List, Select, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/platform/css";
import { Body } from "@/platform/modals/Body";
import { create } from "@/platform/modals/factory";
import { Frame } from "@/platform/modals/Frame";
import { Header } from "@/platform/modals/Header";
import { type Session } from "@/session";
import { Theme } from "@/session/theme";

interface Entry {
  key: Theme.Mode;
  name: string;
  icon: Icon.ReactElement;
}

const OPTIONS: Entry[] = [
  { key: "light", name: "Light", icon: <Icon.LightMode /> },
  { key: "dark", name: "Dark", icon: <Icon.DarkMode /> },
  { key: "system", name: "System", icon: <Icon.Sync /> },
];

const listItem = Component.renderProp(
  (props: List.ItemProps<Theme.Mode>): ReactElement | null => {
    const entry = List.useItem<Theme.Mode, Entry>(props.itemKey);
    if (entry == null) return null;
    return (
      <Select.ListItem {...props} align="center" gap="medium">
        {entry.icon}
        <Text.Text>{entry.name}</Text.Text>
      </Select.ListItem>
    );
  },
);

const Content = ({
  close,
}: Session.Modals.ContentProps<Record<never, never>, void>): ReactElement => {
  const mode = Theme.useSelectMode();
  const dispatch = useDispatch<Dispatch<Theme.Action>>();
  const handleChange = useCallback(
    (next: Theme.Mode) => {
      dispatch(Theme.set(next));
      close();
    },
    [dispatch, close],
  );
  const { data, getItem } = List.useStaticData<Theme.Mode, Entry>({ data: OPTIONS });
  return (
    <Frame className={CSS.B("theme-modal")} bordered rounded="small" pack>
      <Header hideClose icon={<Icon.DarkMode />}>
        Color theme
      </Header>
      <Body pack>
        <Select.Frame<Theme.Mode, Entry>
          data={data}
          getItem={getItem}
          allowNone
          onChange={handleChange}
          initialHover={data.indexOf(mode)}
        >
          <List.Items<Theme.Mode>>{listItem}</List.Items>
        </Select.Frame>
      </Body>
    </Frame>
  );
};

export const useModal = create(Content);
