// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/version/Info.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Logo } from "@synnaxlabs/lyra/logo";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Text } from "@synnaxlabs/lyra/text";

import { CSS } from "@/platform/css";
import { License } from "@/platform/license";
import { Modals } from "@/platform/modals";
import { Update } from "@/platform/version/Update";
import { Session } from "@/session";

const Info = () => {
  const version = Session.Version.use();
  return (
    <Modal.Frame className={CSS.B("version-info")}>
      <Modal.Header icon={<Icon.Info />}>About.Version</Modal.Header>
      <Flex.Box
        className={CSS.BE("version-info", "content")}
        align="center"
        y
        gap="large"
      >
        <Flex.Box y gap="small" justify="center" align="center">
          <a href="https://synnaxlabs.com" target="_blank" rel="noreferrer">
            <Logo.Logo variant="title" className={CSS.BE("version-info", "logo")} />
          </a>
          <Text.Text level="h3" weight={350}>
            Console v{version}
          </Text.Text>
        </Flex.Box>
        <Update />
        <License.Details />
        <Text.Text
          className={CSS.BE("version-info", "footer-note")}
          level="small"
          color={10}
          weight={350}
        >
          © 2022-2026 Synnax Labs, Inc. All rights reserved
        </Text.Text>
      </Flex.Box>
    </Modal.Frame>
  );
};

export const useInfoModal = Modals.create(Info);
