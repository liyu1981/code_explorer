import { Badge } from "../../_components/badge";

interface DefaultBadgeProps {
  isDefault: boolean;
}

export function DefaultBadge({ isDefault }: DefaultBadgeProps) {
  if (!isDefault) return null;
  return (
    <Badge
      variant="outline"
      className="ml-2 py-0 px-1.5 h-4 text-[7px] text-muted-foreground/60 lowercase"
    >
      Default
    </Badge>
  );
}
