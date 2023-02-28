import { ComponentPropsWithoutRef } from "react";

import clsx from "clsx";

import "./Video.css";

export interface VideoProps extends ComponentPropsWithoutRef<"video"> {
  href: string;
}

export const Video = ({ href, className, ...props }: VideoProps): JSX.Element => (
  <video className={clsx("pluto-video", className)} {...props}>
    <source src={href} type="video/mp4" />
  </video>
);
