import { useEffect, useRef, useState } from "react";
import { listSubscriptions } from "../lib/library";
import { subscriptionLink } from "../lib/subscriptionLinks";
import { Icon } from "./Icon";

/** Copies a link that subscribes the reader to every feed of this library. */
export function ShareLibraryButton() {
  const [copied, setCopied] = useState(false);
  const [link, setLink] = useState<string | null>(null);
  const timer = useRef<number | undefined>(undefined);

  useEffect(() => () => window.clearTimeout(timer.current), []);

  const share = async () => {
    const feedUrls = listSubscriptions().map((series) => series.feed_url);
    const url = subscriptionLink(feedUrls, `${window.location.origin}${import.meta.env.BASE_URL}`);
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      window.clearTimeout(timer.current);
      timer.current = window.setTimeout(() => setCopied(false), 2000);
    } catch {
      // The clipboard can be unavailable: show the link so it can be copied by hand.
      setLink(url);
    }
  };

  return (
    <div className="share-library">
      <button type="button" className="btn btn-secondary btn-small" onClick={() => void share()}>
        <Icon name={copied ? "check" : "copy"} size={16} />
        <span>{copied ? "Link copied" : "Share library"}</span>
      </button>
      {link ? (
        <input
          className="add-form-input share-library-link"
          aria-label="Share link"
          readOnly
          value={link}
          onFocus={(event) => event.currentTarget.select()}
        />
      ) : null}
    </div>
  );
}
