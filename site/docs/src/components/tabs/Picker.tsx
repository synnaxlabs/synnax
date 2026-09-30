// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/components/tabs/Picker.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement, useId } from "react";

import { PICKER_ATTRIBUTE, QUERY_ATTRIBUTE } from "@/components/tabs/sync";

export interface Option {
  key: string;
  name: string;
  icon: ReactElement;
}

export interface PickerProps {
  /** The query key the picker shares with the tab blocks on the page. */
  query: string;
  options: Option[];
}

const content = ({ icon, name }: Option): ReactElement => (
  <>
    {icon}
    {name}
  </>
);

/**
 * Renders a menu of choices for a query key on the server. The tabs script in
 * `@/components/tabs/sync` applies a choice to every block that shares the key.
 */
export const Picker = ({ query, options }: PickerProps): ReactElement => {
  const menu = useId();
  return (
    <Flex.Box
      className="docs-picker"
      direction="y"
      empty
      {...{ [QUERY_ATTRIBUTE]: query, [PICKER_ATTRIBUTE]: "" }}
    >
      <Button.Button
        className="docs-picker__trigger"
        variant="outlined"
        justify="between"
        full="x"
        popoverTarget={menu}
      >
        <Flex.Box direction="x" align="center" gap="small" data-tabs-value>
          {content(options[0])}
        </Flex.Box>
        <Icon.Caret.Down />
      </Button.Button>
      <Flex.Box
        id={menu}
        className="docs-picker__menu styled-scrollbar"
        popover="auto"
        direction="y"
        bordered
        rounded
        empty
      >
        {options.map((option, i) => (
          <Button.Button
            key={option.key}
            role="option"
            aria-selected={i === 0}
            data-tab-key={option.key}
            variant="text"
            justify="start"
            full="x"
          >
            {content(option)}
          </Button.Button>
        ))}
      </Flex.Box>
    </Flex.Box>
  );
};
