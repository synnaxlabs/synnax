import { Input, Space } from "@synnaxlabs/pluto";
import "./index.css";
import { Button } from "@synnaxlabs/pluto";
import Logo from "../../lib/Logo/Logo";
import { useNavigate } from "react-router-dom";

export default function Index() {
  const navigate = useNavigate();
  return (
    <Space className="login__container" size={7} justify="center">
      <Logo className="login__logo" variant="icon" />
      <Space>
        <Input name="username" label="Username" />
        <Input name="password" label="Password" type="password" />
      </Space>
      <Button
        className="login__btn"
        onClick={() => {
          navigate("/");
        }}
      >
        Log In
      </Button>
    </Space>
  );
}
