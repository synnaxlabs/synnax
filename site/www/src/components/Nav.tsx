// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useCallback, useRef, useState } from "react";

interface ProductItem {
  icon: Icon.FC;
  title: string;
  description: string;
  href: string;
  section: number;
}

const PRODUCTS: ProductItem[] = [
  {
    icon: Icon.Visualize,
    title: "Visualize and operate",
    description: "Operator dashboards and real-time monitoring interfaces",
    href: "/#visualize",
    section: 2,
  },
  {
    icon: Icon.Arc,
    title: "Automate and control",
    description: "Process control, safety interlocks, and test automation",
    href: "/#automate",
    section: 3,
  },
  {
    icon: Icon.Analyze,
    title: "Review and analyze",
    description: "Post-test analysis, data comparison, and trend review",
    href: "/#review",
    section: 4,
  },
  {
    icon: Icon.Acquire,
    title: "Stream and process",
    description: "Real-time data pipelines, alerting, and live streaming",
    href: "/#stream",
    section: 5,
  },
  {
    icon: Icon.Hardware,
    title: "Device integrations",
    description: "OPC UA, Modbus, EtherCAT, NI, Dewesoft, and more",
    href: "/#integrations",
    section: 6,
  },
  {
    icon: Icon.Terminal,
    title: "Extend with SDKs",
    description: "Python, TypeScript, and C++ clients for custom workflows",
    href: "/#sdks",
    section: 7,
  },
];

const CLOSE_DELAY = 150;

export const Nav = (): ReactElement => {
  const [open, setOpen] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleEnter = useCallback(() => {
    if (timeoutRef.current != null) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    setOpen(true);
  }, []);

  const handleLeave = useCallback(() => {
    timeoutRef.current = setTimeout(() => setOpen(false), CLOSE_DELAY);
  }, []);

  return (
    <Flex.Box direction="x" className="nav-links" align="center" gap={2}>
      <Flex.Box
        className="nav-dropdown-wrap"
        onMouseEnter={handleEnter}
        onMouseLeave={handleLeave}
      >
        <Button.Button variant="text" className="nav-link" href="/#visualize">
          Product
        </Button.Button>
        {open && (
          <Flex.Box
            className="product-dropdown"
            direction="y"
            bordered
            rounded={1}
            background={1}
          >
            <Flex.Box className="product-grid" wrap gap={0}>
              {PRODUCTS.map(({ icon: ItemIcon, title, description, href, section }) => (
                <a key={title} className="product-card" href={href}>
                  <Flex.Box direction="x" gap={3} align="start">
                    <Flex.Box
                      className="product-card__icon-wrap"
                      align="center"
                      justify="center"
                    >
                      <ItemIcon className="product-card__icon" />
                    </Flex.Box>
                    <Flex.Box direction="y" gap={1}>
                      <Text.Text level="p" className="product-card__title">
                        <span className="product-card__section">{section} -&gt; </span>
                        {title}
                      </Text.Text>
                      <Text.Text level="small" className="product-card__desc">
                        {description}
                      </Text.Text>
                    </Flex.Box>
                  </Flex.Box>
                </a>
              ))}
            </Flex.Box>
          </Flex.Box>
        )}
      </Flex.Box>
      <Button.Button variant="text" className="nav-link" href="/company">
        Company
      </Button.Button>
      <Divider.Divider y color={4} className="nav-divider" />
      <Button.Button
        variant="text"
        className="nav-link"
        href="https://docs.synnaxlabs.com/reference"
      >
        Docs
      </Button.Button>
      <Button.Button
        variant="text"
        className="nav-link"
        href="https://docs.synnaxlabs.com/blog"
      >
        Blog
      </Button.Button>
    </Flex.Box>
  );
};
