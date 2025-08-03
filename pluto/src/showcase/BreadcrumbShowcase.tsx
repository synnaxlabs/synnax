// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Breadcrumb } from "@/breadcrumb";
import { Flex } from "@/flex";
import { Icon } from "@/icon";

import { PADDING_STYLE } from "./constants";

export const BreadcrumbShowcase = () => {
  const segments = [
    <Breadcrumb.Segment key="hardware">
      <Icon.Hardware />
      Hardware
    </Breadcrumb.Segment>,
    <Breadcrumb.Segment key="devices" href="https://www.google.com">
      Devices
    </Breadcrumb.Segment>,
    <Breadcrumb.Segment key="labjack" href="https://www.google.com">
      <Icon.Logo.LabJack />
      LabJack T7
    </Breadcrumb.Segment>,
  ];
  const URL = "https://docs.synnaxlabs.com/reference/cluster/cli-reference";
  return (
    <Flex.Box y style={PADDING_STYLE} bordered rounded={1}>
      <Breadcrumb.Breadcrumb level="h4">{segments}</Breadcrumb.Breadcrumb>
      <Breadcrumb.Breadcrumb level="h5">{segments}</Breadcrumb.Breadcrumb>
      <Breadcrumb.Breadcrumb>{segments}</Breadcrumb.Breadcrumb>
      <Breadcrumb.Breadcrumb>
        {Breadcrumb.mapURLSegments(URL, ({ href, segment }) => (
          <Breadcrumb.Segment key={segment} href={href}>
            {segment}
          </Breadcrumb.Segment>
        ))}
      </Breadcrumb.Breadcrumb>
    </Flex.Box>
  );
};
