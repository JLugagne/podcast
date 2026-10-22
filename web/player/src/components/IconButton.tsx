import type { ButtonHTMLAttributes } from "react";
import { Icon, type IconName } from "./Icon";

export type IconButtonVariant = "plain" | "surface" | "primary";
export type IconButtonSize = "sm" | "md" | "lg";

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  label: string;
  icon: IconName;
  variant?: IconButtonVariant;
  size?: IconButtonSize;
}

const ICON_SIZES: Record<IconButtonSize, number> = { sm: 18, md: 22, lg: 30 };

export function IconButton({
  label,
  icon,
  variant = "plain",
  size = "md",
  className,
  ...props
}: IconButtonProps) {
  const classes = ["icon-button", `icon-button-${variant}`, `icon-button-${size}`, className ?? ""]
    .filter(Boolean)
    .join(" ");

  return (
    <button type="button" className={classes} aria-label={label} title={label} {...props}>
      <Icon name={icon} size={ICON_SIZES[size]} />
    </button>
  );
}
