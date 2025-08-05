import { Breadcrumb as B } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";

export interface BreadcrumbProps {
  url: string;
}

export const Breadcrumb = ({ url }: BreadcrumbProps) => (
  <B.Breadcrumb>
    {B.mapURLSegments(url.slice(1), ({ segment, href, index }) => (
      <B.Segment href={`/${href}`} key={index}>
        {segment.split("-").map(caseconv.capitalize).join(" ")}
      </B.Segment>
    ))}
  </B.Breadcrumb>
);
