// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Align } from "@synnaxlabs/pluto/align";
import { Header } from "@synnaxlabs/pluto/header";
import { Text } from "@synnaxlabs/pluto/text";

import "./GetStartedChip.css";

import { ReactElement } from "react";

export interface GetStartedChipProps {
  title: string;
  icon: ReactElement;
  description: string;
  href: string;
}

export const GetStartedChip = ({
  icon,
  title,
  description,
  href,
}: GetStartedChipProps): ReactElement => {
  return (
    <a
      href={href}
      className="docs-get-started-chip"
      style={{
        textDecoration: "none",
      }}
    >
      <Header.Header level="h2" className="pluto--bordered" wrap>
        <Align.Space 
        className="text-container"
        empty style={{ minWidth: 100, flex: "1 1 100px" }}>
          <Header.Title>{title}</Header.Title>
          <Header.Title level="p">{description}</Header.Title>
        </Align.Space>
      </Header.Header>
    </a>
  );
};
