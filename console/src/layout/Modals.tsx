import { Icon } from "@synnaxlabs/media";
import { Button, Modal, Nav, Text } from "@synnaxlabs/pluto";
import { CSSProperties } from "react";

import { Layout } from "@/layout";
import { Content } from "@/layout/Content";
import { WindowProps } from "@/layout/layout";
import { useSelectModals } from "@/layout/selectors";

const layoutCSS = (window?: WindowProps): CSSProperties => ({
  width: window?.size?.width,
  height: window?.size?.height,
  minWidth: window?.minSize?.width,
  minHeight: window?.minSize?.height,
});

export const Modals = () => {
  const layouts = useSelectModals();
  const remove = Layout.useRemover();
  return (
    <>
      {layouts.map(({ key, name, window }) => (
        <Modal.Modal
          key={key}
          visible
          close={() => remove(key)}
          style={layoutCSS(window)}
        >
          {window?.navTop && (
            <Nav.Bar location="top" size="6rem">
              {(window?.showTitle ?? true) && (
                <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
                  <Text.Text level="p" shade={6} weight={450}>
                    {name}
                  </Text.Text>
                </Nav.Bar.Start>
              )}
              <Nav.Bar.End style={{ paddingRight: "1rem" }}>
                <Button.Icon onClick={() => remove(key)} size="small">
                  <Icon.Close style={{ color: "var(--pluto-gray-l8)" }} />
                </Button.Icon>
              </Nav.Bar.End>
            </Nav.Bar>
          )}
          <Content layoutKey={key} />
        </Modal.Modal>
      ))}
    </>
  );
};
