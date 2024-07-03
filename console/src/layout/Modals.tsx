import { Icon } from "@synnaxlabs/media";
import { Button, Menu, Modal as Core, Nav, Text } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { CSSProperties, FC, ReactElement } from "react";

import { Content } from "@/layout/Content";
import { useRemover } from "@/layout/hooks";
import { State, WindowProps } from "@/layout/layout";
import { useSelectModals } from "@/layout/selectors";
import { DefaultContextMenu } from "@/layout/Window";

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

const BreadCrumb = ({ name, icon }: Pick<State, "name" | "icon">): ReactElement => {
  let iconC: ReactElement | undefined = undefined;
  if (icon) {
    const IconC = deep.get<FC, typeof Icon>(Icon, icon);
    iconC = <IconC />;
  }
  const split = name.split(".");
  const content: (ReactElement | string)[] = split
    .map((name, index) => [
      <Icon.Caret.Right
        key={`${name}-${index}`}
        style={{
          transform: "scale(0.8) translateY(1px)",
          color: "var(--pluto-gray-l6)",
        }}
      />,
      name,
    ])
    .flat();
  return (
    <Text.WithIcon level="p" shade={7} weight={450} size={0.5}>
      {iconC}
      {...content}
    </Text.WithIcon>
  );
};

const Modal = ({ state, remove }: ModalProps) => {
  const { key, name, window, icon } = state;
  const menuProps = Menu.useContextMenu();
  return (
    <Menu.ContextMenu menu={() => <DefaultContextMenu />} {...menuProps}>
      <Core.Modal key={key} visible close={() => remove(key)} style={layoutCSS(window)}>
        {window?.navTop && (
          <Nav.Bar location="top" size="6rem">
            {(window?.showTitle ?? true) && (
              <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
                <BreadCrumb name={name} icon={icon} />
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
    </Menu.ContextMenu>
  );
};

export const Modals = () => {
  const layouts = useSelectModals();
  const remove = useRemover();
  return (
    <>
      {layouts.map((l) => l.window && <Modal key={l.key} state={l} remove={remove} />)}
    </>
  );
};
