import { usePresentationContext } from "../../components";
import { Space, Text } from "@synnaxlabs/pluto";
import {
  cloneElement,
  PropsWithChildren,
  ReactComponentElement,
  ReactElement,
} from "react";
import { motion } from "framer-motion";
import "./BulletSlide.css";
import clsx from "clsx";

type BulletProps = PropsWithChildren<{
  bullet?: React.ReactElement;
  number?: number;
  visible?: boolean;
  subText?: string;
}>;

export const Bullet = ({
  children,
  bullet,
  subText,
  number,
  visible = false,
}: BulletProps) => {
  return visible ? (
    <motion.div
      style={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 1 }}
    >
      <Space direction="horizontal" justify="end" align="center" size="large">
        {bullet ? (
          bullet
        ) : (
          <Text level="h1" className="bullet__number">
            {number}
          </Text>
        )}
        <div style={{ position: "relative", top: subText && 12 }}>
          {typeof children === "string" ? (
            <Text className="bullet__text" level="h2">
              {children}
            </Text>
          ) : (
            children
          )}
          {subText && (
            <Text level="h4" className="bullet__subtext">
              {subText}
            </Text>
          )}
        </div>
      </Space>
    </motion.div>
  ) : null;
};

type BulletSlideProps = {
  title?: string | ReactElement;
  theme?: "outlined" | "primary";
  extra?: ReactElement;
  children: ReactComponentElement<typeof Bullet, BulletProps>[];
};

export default function BulletSlide({
  children,
  theme = "outlined",
  extra,
  title,
}: BulletSlideProps) {
  const { transition } = usePresentationContext();
  return (
    <Space
      className={`bullet-slide__container--${theme}`}
      direction="vertical"
      justify="center"
      align="center"
      size={12}
      grow
    >
      {title &&
        (typeof title === "string" ? (
          <Text level="h1" className={`bullet-slide__title`}>
            {title}
          </Text>
        ) : (
          title
        ))}
      {extra}
      <Space align="start" size={8}>
        {children.map((child, index) =>
          cloneElement(child, {
            visible: title ? index <= transition - 1 : index <= transition,
            number: index + 1,
          })
        )}
      </Space>
    </Space>
  );
}
