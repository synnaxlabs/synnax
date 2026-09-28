// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type CSSProperties, type PropsWithChildren, type ReactElement } from "react";

import { Panel } from "@/ui/Panel";

export interface TableProps extends PropsWithChildren {
  /** columns is the grid-template-columns every row shares. */
  columns: string;
  head: string[];
}

/** Table lays rows out on one shared grid with a header row above them. */
export const Table = ({ columns, head, children }: TableProps): ReactElement => (
  <Panel className="portal-list">
    <Flex.Box
      className="portal-list__row portal-list__head"
      style={{ gridTemplateColumns: columns }}
    >
      {head.map((h) => (
        <Text.Text key={h} level="small" weight={500} color={9}>
          {h}
        </Text.Text>
      ))}
    </Flex.Box>
    {children}
  </Panel>
);

export interface RowProps extends PropsWithChildren {
  columns: string;
  /** href makes the row a link, with a caret at its end. */
  href?: string;
  style?: CSSProperties;
}

const ROW_STYLE: CSSProperties = { height: "auto" };

/** Row is one line of a {@link Table}. */
export const Row = ({ columns, href, style, children }: RowProps): ReactElement =>
  href == null ? (
    <Flex.Box
      className="portal-list__row"
      style={{ ...ROW_STYLE, gridTemplateColumns: columns, ...style }}
    >
      {children}
    </Flex.Box>
  ) : (
    <Button.Button
      href={href}
      variant="text"
      className="portal-list__row"
      style={{ ...ROW_STYLE, gridTemplateColumns: columns, ...style }}
    >
      {children}
      <Icon.Caret.Right color={8} className="portal-list__caret" />
    </Button.Button>
  );
