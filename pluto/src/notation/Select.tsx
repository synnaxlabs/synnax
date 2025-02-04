// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type KeyedNamed } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type Notation } from "@/notation/notation";
import { Button, type ButtonProps } from "@/select/Button";

const DATA: KeyedNamed<Notation>[] = [
  { key: "standard", name: "Standard" },
  { key: "scientific", name: "Scientific" },
  { key: "engineering", name: "Engineering" },
];

export interface SelectNotationProps
  extends Omit<
    ButtonProps<Notation, KeyedNamed<Notation>>,
    "data" | "entryRenderKey"
  > {}

export const Select = ({ ...props }: SelectNotationProps): ReactElement => (
  <Button {...props} entryRenderKey="name" data={DATA} />
);
