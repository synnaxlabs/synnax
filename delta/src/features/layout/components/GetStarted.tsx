// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Logo } from "@synnaxlabs/media";
import { Text, Button, Space } from "@synnaxlabs/pluto";

import { VisCreateButton } from "@/features/vis";

import "./GetStarted.css";

export const GetStarted = (): JSX.Element => {
  return (
    <Space.Centered className="delta-get-started" align="center" size={6}>
      <Logo variant="title" className="delta-get-started__logo" />
      <Text level="h1">Get Started</Text>
      <Space direction="x" size="large" justify="center" wrap>
        <Button startIcon={<Icon.Cluster />} size="large">
          Connect a Cluster
        </Button>
        <VisCreateButton size="large" />
      </Space>
      <Text.Link href="https://docs.synnaxlabs.com" target="_blank" level="h4">
        Read the Documentation
      </Text.Link>
    </Space.Centered>
  );
};
