import type { MiddlewareHandler } from "astro";

export const onRequest: MiddlewareHandler = async (_, next) => {
  const response = await next();

  response.headers.set(
    "Content-Security-Policy",
    "default-src 'self'; script-src 'self' 'unsafe-inline' https://vercel.live https://us-assets.i.posthog.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com data:; connect-src 'self' https://ywd9t0jxcs-dsn.algolia.net https://raw.githubusercontent.com https://us.i.posthog.com https://formspree.io; img-src 'self' data: https://us-assets.i.posthog.com https://synnax.nyc3.cdn.digitaloceanspaces.com; media-src 'self' https://synnax.nyc3.cdn.digitaloceanspaces.com; frame-src https://vercel.live https://www.youtube.com http://localhost:4321; object-src 'none';",
  );

  return response;
};
