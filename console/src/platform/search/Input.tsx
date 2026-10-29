// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Input as Base } from "@synnaxlabs/lyra/input";
import { type ReactElement } from "react";

export interface InputProps
  extends Pick<Base.TextProps, "value" | "onChange" | "placeholder"> {}

const startContent = <Icon.Search color={9} />;

/** A quiet text input with a leading search icon. */
export const Input = (props: InputProps): ReactElement => (
  <Base.Text
    size="small"
    level="h5"
    color={9}
    startContent={startContent}
    variant="text"
    {...props}
  />
);
