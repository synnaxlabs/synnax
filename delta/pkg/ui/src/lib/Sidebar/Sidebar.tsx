import { Space } from "@arya-analytics/pluto";
import Logo from "../Logo/Logo";
import "./Sidebar.css";
import { ThemeSwitch } from "@arya-analytics/pluto";

export default function Sidebar() {
  return (
    <Space className="sidebar__container" justify="spaceBetween">
      <Space className="sidebar__header">
        <Logo className="sidebar__logo" variant="icon" />
      </Space>
      <Space className="sidebar__body"></Space>
      <Space className="sidebar__footer" align="center">
        <ThemeSwitch />
      </Space>
    </Space>
  );
}
