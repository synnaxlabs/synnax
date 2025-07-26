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
}: BreadcrumbProps) => {
  const breadcrumbSegments: Core.Segments = [{ label: name, weight: 450, shade: 10 }];
  if (parent != null && showParent)
    breadcrumbSegments.push({ label: parent.name, weight: 400, shade: 8 });
  return <Core.Breadcrumb {...rest}>{breadcrumbSegments}</Core.Breadcrumb>;
};
