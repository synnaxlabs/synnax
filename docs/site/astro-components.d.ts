// src/types/astro-components.d.ts
declare module "*.astro" {
  import type { AstroComponentFactory } from "astro/runtime";
  const component: AstroComponentFactory;
  export default component;
}
