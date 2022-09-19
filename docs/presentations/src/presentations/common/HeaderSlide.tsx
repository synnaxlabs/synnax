import { Header, Space } from "@synnaxlabs/pluto";
import { PropsWithChildren } from "react";
export type HeaderProps = PropsWithChildren<{
  title: string;
  color: string;
  textColor?: string;
}>;

export default function HeaderSlide({ color, title, children, textColor }: HeaderProps) {
  return (
    <Space grow>
      {children}
      <Header
        level="h1"
        text={title}
        style={{ backgroundColor: color, border: "none"}}
        textColor={textColor}
      />
    </Space>
  );
}
