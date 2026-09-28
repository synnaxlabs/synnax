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
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useCallback } from "react";

import { Mark } from "@/ui/shell/Mark";

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
  const handleChange = useCallback((key: string | null) => {
    if (key != null) window.location.assign(`/?org=${key}`);
  }, []);
  const label = (
    <>
      <Mark name={selected.name} />
      <Text.Text level="p" weight={500} overflow="ellipsis">
        {selected.name}
      </Text.Text>
    </>
  );
  if (options.length === 1) return <span className="portal-scope">{label}</span>;
  return (
    <Dialog.Frame variant="floating" location={{ x: "left", y: "bottom" }}>
      <Select.Frame<string, undefined> value={selected.key} onChange={handleChange}>
        <Dialog.Trigger
          size="large"
          variant="text"
          hideCaret
          className="portal-scope"
          aria-label="Switch scope"
        >
          {label}
          <Icon.Caret.Down className="portal-scope__caret" />
        </Dialog.Trigger>
        <Select.Dialog className="portal-scope__dialog" background={1} rounded>
          <Select.List full="x">
            {options.map((o) => (
              <Select.Item key={o.key} itemKey={o.key} className="portal-scope__item">
                <Mark name={o.name} />
                <Text.Text level="p" overflow="ellipsis">
                  {o.name}
                </Text.Text>
              </Select.Item>
            ))}
          </Select.List>
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
