import { useI18n } from "../i18n";
import { usePlayback } from "../playback/PlaybackProvider";
import { formatRate, PLAYBACK_RATES } from "../playback/rates";
import { Icon } from "./Icon";
import { Menu } from "./Menu";

export function SpeedMenu() {
  const { t } = useI18n();
  const { rate, setRate } = usePlayback();

  return (
    <Menu
      label={t("playbackSpeed", { rate: formatRate(rate) })}
      className="menu-inline"
      trigger={
        <>
          <Icon name="speed" size={18} />
          <span>{formatRate(rate)}</span>
        </>
      }
    >
      {(close) =>
        PLAYBACK_RATES.map((value) => (
          <button
            key={value}
            type="button"
            role="menuitemradio"
            aria-checked={value === rate}
            className="menu-item"
            onClick={() => {
              setRate(value);
              close();
            }}
          >
            <span className="menu-check" aria-hidden="true">
              {value === rate ? <Icon name="check" size={16} /> : null}
            </span>
            <span>{formatRate(value)}</span>
          </button>
        ))
      }
    </Menu>
  );
}
