export function ErrorBanner({
  message,
  title = "Something went wrong",
}: {
  message: string;
  title?: string;
}) {
  return (
    <div className="banner banner-error" role="alert">
      <span className="banner-icon" aria-hidden="true">
        !
      </span>
      <div className="banner-body">
        <span className="banner-title">{title}</span>
        <span>{message}</span>
      </div>
    </div>
  );
}
