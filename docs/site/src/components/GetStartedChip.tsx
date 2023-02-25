import { Text, Header, HeaderProps, Space } from "@synnaxlabs/pluto";
import { IoMdArrowDropright } from "react-icons/io";

export interface GetStartedChipProps {
  title: string;
  icon: JSX.Element;
  description: string;
  href: string;
}

export const GetStartedChip = ({
  icon,
  title,
  description,
  href,
}: GetStartedChipProps): JSX.Element => {
  return (
    <a
      href={href}
      style={{
        textDecoration: "none",
      }}
    >
      <Header
        level="h2"
        className="pluto-bordered"
        style={{
          padding: "1rem 2rem",
          borderRadius: "var(--pluto-border-radius)",
        }}
      >
        <Space empty>
          <Header.Title startIcon={icon}>{title}</Header.Title>
          <Header.Title level="p">{description}</Header.Title>
        </Space>
        <Text.WithIcon
          level="h4"
          color="var(--pluto-primary-p1)"
          style={{ whiteSpace: "nowrap !important" }}
          endIcon={<IoMdArrowDropright />}
          empty
        >
          Read More
        </Text.WithIcon>
      </Header>
    </a>
  );
};
