import { ComponentPropsWithoutRef } from "react";

import { CSS } from "@/css";

import "./Video.css";

export interface VideoProps extends ComponentPropsWithoutRef<"video"> {
  href: string;
}

export const Video = ({ href, className, ...props }: VideoProps): JSX.Element => (
  <video className={CSS(CSS.B("video"), className)} {...props}>
    <source src={href} type="video/mp4" />
  </video>
);
