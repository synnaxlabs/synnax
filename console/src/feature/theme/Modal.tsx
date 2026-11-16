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
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { type Option, OPTIONS } from "@/feature/theme/options";
import { CSS } from "@/platform/css";
import { create } from "@/platform/modals/factory";
import { type Session } from "@/session";
import { Theme } from "@/session/theme";

const ListItem = (props: List.ItemProps<Theme.Mode>): ReactElement | null => {
  const entry = List.useItem<Theme.Mode, Option>(props.itemKey);
  if (entry == null) return null;
  return (
    <Select.Item {...props} align="center" gap="medium">
      {entry.icon}
      {entry.name}
    </Select.Item>
  );
};

const listItem = Component.renderProp(ListItem);

const Content = ({
  close,
}: Session.Modals.ContentProps<Record<never, never>, void>): ReactElement => {
  const mode = Theme.useSelectMode();
  const dispatch = useDispatch<Dispatch<Theme.Action>>();
  const handleChange = (next: Theme.Mode) => {
    dispatch(Theme.set(next));
    close();
  };
  const { data, getItem } = List.useStaticData<Theme.Mode, Option>({ data: OPTIONS });
  return (
    <Modal.Frame className={CSS.B("theme-modal")} bordered rounded="small" pack>
      <Modal.Header closeHidden icon={<Icon.DarkMode />}>
        Color theme
      </Modal.Header>
      <Modal.Body pack>
        <Select.Frame<Theme.Mode, Option>
          data={data}
          getItem={getItem}
          allowNone
          onChange={handleChange}
          initialHover={data.indexOf(mode)}
        >
          <List.Scroll>
            <List.Items<Theme.Mode>>{listItem}</List.Items>
          </List.Scroll>
        </Select.Frame>
      </Modal.Body>
    </Modal.Frame>
  );
};

export const useModal = create(Content);
