// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { type Extractors } from "@/export/extractor";

const [Context, useContext] = context.create<Extractors>({
  displayName: "Export.Context",
  providerName: "Export.ExtractorsProvider",
});

export const useExtractors = (): Extractors => useContext("Export.useExtractors");

export interface ExtractorsProviderProps extends PropsWithChildren {
  extractors: Extractors;
}

export const ExtractorsProvider = ({
  extractors,
  ...rest
}: ExtractorsProviderProps): ReactElement => <Context value={extractors} {...rest} />;
