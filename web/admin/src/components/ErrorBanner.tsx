import type { ReactNode } from "react";

export interface ErrorBannerProps {
  message: string;
  title?: string;
  details?: string[];
}

export function ErrorBanner({
  message,
  title = "Something went wrong",
  details,
}: ErrorBannerProps) {
  return (
    <div className="banner banner-error" role="alert">
      <span className="banner-icon" aria-hidden="true">
        !
      </span>
      <div className="banner-body">
        <span className="banner-title">{title}</span>
        <span>{message}</span>
        {details && details.length > 0 ? (
          <ul className="banner-list">
            {details.map((detail) => (
              <li key={detail}>{detail}</li>
            ))}
          </ul>
        ) : null}
      </div>
    </div>
  );
}

export interface InfoBannerProps {
  message: ReactNode;
  title?: string;
}

export function InfoBanner({ message, title }: InfoBannerProps) {
  return (
    <div className="banner banner-info" role="status">
      <span className="banner-icon" aria-hidden="true">
        i
      </span>
      <div className="banner-body">
        {title ? <span className="banner-title">{title}</span> : null}
        <span>{message}</span>
      </div>
    </div>
  );
}
