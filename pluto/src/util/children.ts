// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Children, ReactElement } from "react";

export const reactElementToArray = <
  P = any,
  T extends string | React.JSXElementConstructor<any> =
    | string
    | React.JSXElementConstructor<any>
>(
  children: ReactElement<P, T> | Array<ReactElement<P, T>>
): Array<ReactElement<P, T>> => Children.toArray(children) as Array<ReactElement<P, T>>;
