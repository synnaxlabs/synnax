// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Text } from "@synnaxlabs/pluto";

import { useLayoutPlacer } from "@/features/layout";

export const ReadTheDocsButton = (): JSX.Element => {
  const placer = useLayoutPlacer();
  const handleClick = (): void => {
    placer({
      key: "docs",
      type: "docs",
      location: "mosaic",
      name: "Documentation",
      tab: { editable: false },
    });
  };

  return (
    <Text.Link target="_blank" level="h4" onClick={handleClick}>
      Read the Documentation
    </Text.Link>
  );
};
