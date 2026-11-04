// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/port/Rows.css";

import { Text } from "@synnaxlabs/lyra/text";
import { type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/platform/css";

/**
 * Rows of ports in as many columns as fit, read down. Every row's cells start at the
 * same place, after its label.
 */
export const Rows = ({ children }: PropsWithChildren): ReactElement => (
  <div className={CSS.B("port-rows")}>{children}</div>
);

export interface RowProps extends PropsWithChildren {
  label: string;
}

/** One bank or port of {@link Rows}: its label, then its grid. */
export const Row = ({ label, children }: RowProps): ReactElement => (
  <div className={CSS.BE("port-rows", "row")}>
    <Text.Text level="small" color={9} className={CSS.BE("port-rows", "label")}>
      {label}
    </Text.Text>
    {children}
  </div>
);
