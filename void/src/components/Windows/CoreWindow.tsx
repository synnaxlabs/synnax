import { Nav, Space } from "@synnaxlabs/pluto";
import { PropsWithChildren } from "react";
import TopNavbar from "../../Layouts/Main/TopNavbar";
import "./CoreWindow.css";

export interface FormWindowProps extends PropsWithChildren<any> {}

export default function CoreWindow({ children }: FormWindowProps) {
  return (
    <Space direction="vertical" size="large" className="main__container" empty>
      <TopNavbar />
      {children}
    </Space>
  );
}
