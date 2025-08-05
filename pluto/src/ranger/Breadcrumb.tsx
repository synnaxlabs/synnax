import { type ranger } from "@synnaxlabs/client";

import { Breadcrumb as Core } from "@/breadcrumb";

export interface BreadcrumbProps extends Omit<Core.BreadcrumbProps, "children"> {
  name: string;
  showParent?: boolean;
  parent?: Pick<ranger.Payload, "name"> | null;
}

export const Breadcrumb = ({
  name,
  parent,
  showParent = true,
  ...rest
}: BreadcrumbProps) => (
  <Core.Breadcrumb {...rest}>
    <Core.Segment weight={450} color={10}>
      {name}
    </Core.Segment>
    {parent != null && showParent && (
      <Core.Segment weight={400} color={8}>
        {parent.name}
      </Core.Segment>
    )}
  </Core.Breadcrumb>
);
