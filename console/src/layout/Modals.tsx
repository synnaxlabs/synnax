import { Icon } from "@synnaxlabs/media";
import { Button, Modal as Core, Nav, Text } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { CSSProperties, FC, ReactElement } from "react";

import { Content } from "@/layout/Content";
import { useRemover } from "@/layout/hooks";
import { State, WindowProps } from "@/layout/layout";
import { useSelectModals } from "@/layout/selectors";

const layoutCSS = (window?: WindowProps): CSSProperties => ({
  width: window?.size?.width,
  height: window?.size?.height,
  minWidth: window?.minSize?.width,
  minHeight: window?.minSize?.height,
});

interface ModalProps {
  state: State;
  remove: (key: string) => void;
}

const Modal = ({ state, remove }: ModalProps) => {
  const { key, name, window, icon } = state;
  let iconC: ReactElement | undefined = undefined;
  if (icon) {
    const IconC = deep.get<FC, typeof Icon>(Icon, icon);
    iconC = <IconC />;
  }
  return (
    <Core.Modal key={key} visible close={() => remove(key)} style={layoutCSS(window)}>
      {window?.navTop && (
        <Nav.Bar location="top" size="6rem">
          {(window?.showTitle ?? true) && (
            <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
              <Text.WithIcon level="p" shade={6} weight={450} startIcon={iconC}>
                {name}
              </Text.WithIcon>
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
    </Core.Modal>
  );
};

export const Modals = () => {
  const layouts = useSelectModals();
  const remove = useRemover();
  return (
    <>
      {layouts.map(
        (layout) =>
          layout.window && <Modal key={layout.key} state={layout} remove={remove} />,
      )}
    </>
  );
};
