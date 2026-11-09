// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as Base } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { deep } from "@synnaxlabs/x";

const path = deep.path<{ duration: number }>();

export const Form = () => (
  <Base.Field path={path.duration}>
    {(p) => <Input.TimeSpan inline {...p} />}
  </Base.Field>
);
