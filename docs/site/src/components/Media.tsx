// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Video as Base } from "@synnaxlabs/pluto/video";
import {
  type DetailedHTMLProps,
  type ImgHTMLAttributes,
  type ReactElement,
  useEffect,
  useRef,
  useState,
} from "react";

interface MediaProps {
  id: string;
  themed?: boolean;
}

export interface VideoProps
  extends
    MediaProps,
    Omit<
      DetailedHTMLProps<React.VideoHTMLAttributes<HTMLVideoElement>, HTMLVideoElement>,
      "id"
    > {}

const CDN_ROOT = "https://synnax.nyc3.cdn.digitaloceanspaces.com/docs";

const useLiveTheme = (): string => {
  const [theme, setTheme] = useState(
    window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light",
  );
  useEffect(() => {
    const listener = (e: MediaQueryListEvent) => {
      setTheme(e.matches ? "dark" : "light");
    };
    const bindListener = () => {
      window
        .matchMedia("(prefers-color-scheme: dark)")
        .addEventListener("change", listener);
    };
    bindListener();
    document.addEventListener("astro:after-swap", bindListener);
    return () => {
      window
        .matchMedia("(prefers-color-scheme: dark)")
        .removeEventListener("change", listener);
    };
  }, []);
  return theme;
};

export const Video = ({ id, themed = true, ...rest }: VideoProps): ReactElement => {
  const theme = useLiveTheme();
  const url = `${CDN_ROOT}/${id}${themed ? `-${theme}` : ""}.mp4`;
  const ref = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (ref.current) ref.current.load();
  }, [url]);

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (ref.current == null) return;
        if (entry.isIntersecting) ref.current.play().catch(console.error);
        else ref.current.pause();
      },
      { threshold: 0.85 },
    );
    if (ref.current != null) observer.observe(ref.current);
    return () => {
      if (ref.current != null) observer.unobserve(ref.current);
    };
  }, []);

  return <Base.Video ref={ref} href={url} loop muted {...rest} />;
};

export interface ImageProps
  extends
    MediaProps,
    Omit<
      DetailedHTMLProps<ImgHTMLAttributes<HTMLImageElement>, HTMLImageElement>,
      "id"
    > {
  extension?: "png" | "jpg" | "jpeg" | "webp" | "svg";
}

export const Image = ({
  id,
  themed = true,
  className,
  extension = "png",
  ...rest
}: ImageProps): ReactElement => {
  const theme = useLiveTheme();
  let url = `${CDN_ROOT}/${id}`;
  if (themed) url += `-${theme}`;
  url += `.${extension}`;
  const ref = useRef<HTMLImageElement>(null);
  useEffect(() => {
    if (ref.current) ref.current.src = url;
  }, []);
  return <img src={url} {...rest} />;
};
