// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Text, Header, Space } from "@synnaxlabs/pluto";

export interface GetStartedChipProps {
  title: string;
  icon: JSX.Element;
  description: string;
  href: string;
}

export const GetStartedChip = ({
  icon,
  title,
  description,
  href,
}: GetStartedChipProps): JSX.Element => {
  return (
    <a
      href={href}
      style={{
        textDecoration: "none",
      }}
    >
      <Header
        level="h2"
        className="pluto--bordered"
        style={{
          padding: "1rem 2rem",
          borderRadius: "var(--pluto-border-radius)",
        }}
        wrap
      >
        <Space empty style={{ minWidth: 100, flex: "1 1 100px" }}>
          <Header.Title startIcon={icon}>{title}</Header.Title>
          <Header.Title level="p">{description}</Header.Title>
        </Space>
        <Text.WithIcon
          level="h4"
          color="var(--pluto-primary-p1)"
          style={{ whiteSpace: "nowrap !important" }}
          endIcon={<Icon.Caret.Right />}
          empty
        >
          Read More
        </Text.WithIcon>
      </Header>
    </a>
  );
};
