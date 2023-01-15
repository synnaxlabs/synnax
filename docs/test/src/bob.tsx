import { useEffect } from "react";

export const Hello = () => {
  useEffect(() => {
    console.log(window.document);
  });
  return <h1>Hello</h1>;
};
