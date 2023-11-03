// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentPropsWithoutRef, type ReactElement } from "react";

export const Table = (): ReactElement => {
  return (
    <table>
      <tr>
        <td></td>
      </tr>

      {children}
    </table>
  );
};

export interface TRProps extends ComponentPropsWithoutRef<"tr"> {}

export const TR = (props: TRProps): ReactElement => {
  return <tr {...props} />;
};

export interface TDProps extends ComponentPropsWithoutRef<"td"> {
  children: ReactElement;
}

export const TD = (): ReactElement => {
  return <td />;
};

export interface THProps extends ComponentPropsWithoutRef<"th"> {}

export const TH = (): ReactElement => {};
