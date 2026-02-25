import { Caret, Flex, Icon, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useRef, useState } from "react";

interface ProductItem {
  icon: Icon.FC;
  title: string;
  description: string;
  href: string;
}

const PRODUCTS: ProductItem[] = [
  {
    icon: Icon.Visualize,
    title: "Visualize & Operate",
    description: "Operator dashboards and real-time monitoring interfaces",
    href: "/product/visualize",
  },
  {
    icon: Icon.Control,
    title: "Automate & Control",
    description: "Process control, safety interlocks, and test automation",
    href: "/product/automate",
  },
  {
    icon: Icon.Analyze,
    title: "Review & Analyze",
    description: "Post-test analysis, data comparison, and trend review",
    href: "/product/analyze",
  },
  {
    icon: Icon.Acquire,
    title: "Stream & Process",
    description: "Real-time data pipelines, alerting, and live streaming",
    href: "/product/stream",
  },
  {
    icon: Icon.Hardware,
    title: "Device Integrations",
    description: "OPC UA, Modbus, EtherCAT, NI, Dewesoft, and more",
    href: "/product/integrations",
  },
  {
    icon: Icon.LinkExternal,
    title: "Extend with SDKs",
    description: "Python, TypeScript, and C++ clients for custom workflows",
    href: "/product/sdks",
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
    <Flex.Box direction="x" className="nav-links" align="center" gap={5}>
      <Flex.Box
        className="nav-dropdown-wrap"
        onMouseEnter={handleEnter}
        onMouseLeave={handleLeave}
      >
        <Flex.Box direction="x" align="center" gap={1} className="nav-link-trigger">
          <Text.Text level="h5" className="nav-link">
            Product
          </Text.Text>
          <Caret.Animated
            enabled={open}
            enabledLoc="bottom"
            disabledLoc="left"
            className="nav-caret"
          />
        </Flex.Box>
        {open && (
          <Flex.Box
            className="product-dropdown"
            direction="y"
            bordered
            rounded={1}
            background={1}
          >
            <Flex.Box className="product-grid" wrap gap={0}>
              {PRODUCTS.map(({ icon: ItemIcon, title, description, href }) => (
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
      <Text.Text level="h5" className="nav-link">
        <a href="https://docs.synnaxlabs.com">Docs</a>
      </Text.Text>
      <Text.Text level="h5" className="nav-link">
        <a href="/#company">Company</a>
      </Text.Text>
    </Flex.Box>
  );
};
