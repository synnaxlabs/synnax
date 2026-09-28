// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useCallback } from "react";

export interface ScopeOption {
  key: string;
  name: string;
  personal: boolean;
}

export interface ScopeProps {
  /** options are the scopes the user can act for, personal first. */
  options: ScopeOption[];
  selected: ScopeOption;
}

/**
 * Scope names who the portal acts for. A user on a team gets a menu that opens the
 * overview of another scope.
 */
export const Scope = ({ options, selected }: ScopeProps): ReactElement => {
  const handleChange = useCallback((key: string) => {
    window.location.assign(`/?org=${key}`);
  }, []);
  if (options.length === 1)
    return (
      <Text.Text level="p" weight={500} overflow="ellipsis" className="portal-scope">
        {selected.name}
      </Text.Text>
    );
  return (
    <Dialog.Frame variant="floating" location={{ x: "left", y: "bottom" }}>
      <Dialog.Trigger variant="text" className="portal-scope" aria-label="Switch scope">
        <Text.Text level="p" weight={500} overflow="ellipsis">
          {selected.name}
        </Text.Text>
      </Dialog.Trigger>
      <Dialog.Dialog
        bordered
        rounded
        background={1}
        className="portal-menu"
        style={{ padding: "1rem", minWidth: "28rem" }}
      >
        <Menu.Menu level="small" value={selected.key} onChange={handleChange}>
          {options.map((o) => (
            <Menu.Item key={o.key} itemKey={o.key}>
              {o.personal ? <Icon.User /> : <Icon.Role />}
              {o.name}
            </Menu.Item>
          ))}
        </Menu.Menu>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
