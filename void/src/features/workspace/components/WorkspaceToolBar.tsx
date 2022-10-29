import { Header, Space } from "@synnaxlabs/pluto";
import { AiFillFolder } from "react-icons/ai";
import { MdWorkspacesFilled } from "react-icons/md";

const Content = () => {
  return (
    <Space empty style={{ height: "100%" }}>
      <Header level="h4" divided icon={<MdWorkspacesFilled />}>
        Workspace
      </Header>
    </Space>
  );
};

export const WorkspaceToolBar = {
  key: "workspace",
  icon: <MdWorkspacesFilled />,
  content: <Content />,
};
