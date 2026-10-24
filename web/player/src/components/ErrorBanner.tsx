import { t } from "../i18n";

export function ErrorBanner({ message, title }: { message: string; title?: string }) {
  return (
    <div className="banner banner-error" role="alert">
      <span className="banner-icon" aria-hidden="true">
        !
      </span>
      <div className="banner-body">
        <span className="banner-title">{title ?? t("somethingWentWrong")}</span>
        <span>{message}</span>
      </div>
    </div>
  );
}
