// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Logo } from "@synnaxlabs/media";
import { Text, Space } from "@synnaxlabs/pluto";

import { ConnectClusterButton } from "@/features/cluster";
import { ReadTheDocsButton } from "@/features/docs";
import { CreateVisButton } from "@/features/vis";

import "./GetStarted.css";

export const GetStarted = (): JSX.Element => (
  <Space.Centered className="delta-get-started" align="center" size={6}>
    <Logo variant="title" className="delta-get-started__logo" />
    <Text level="h1">Get Started</Text>
    <Space direction="x" size="large" justify="center" wrap>
      <ConnectClusterButton />
      <CreateVisButton />
    </Space>
    <ReadTheDocsButton />
  </Space.Centered>
);
