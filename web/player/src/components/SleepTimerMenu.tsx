import { useI18n } from "../i18n";
import {
  formatCountdown,
  SLEEP_TIMER_OPTIONS,
  type SleepTimer,
  type SleepTimerOption,
  sleepTimerLabel,
  sleepTimerOptionLabel,
} from "../lib/sleepTimer";
import { usePlayback } from "../playback/PlaybackProvider";
import { Icon } from "./Icon";
import { Menu } from "./Menu";

function isSelected(option: SleepTimerOption, timer: SleepTimer): boolean {
  switch (option) {
    case "off":
      return timer.kind === "off";
    case "chapter":
      return timer.kind === "chapter";
    case "episode":
      return timer.kind === "episode";
    default:
      return timer.kind === "minutes" && timer.minutes === Number(option);
  }
}

export function SleepTimerMenu() {
  const { t } = useI18n();
  const { sleepTimer, sleepRemaining, setSleepTimerOption } = usePlayback();
  const label = sleepTimerLabel(sleepTimer, sleepRemaining);
  const remaining = sleepTimer.kind === "minutes" ? (sleepRemaining ?? sleepTimer.minutes * 60) : 0;
  const triggerLabel =
    sleepTimer.kind === "minutes"
      ? formatCountdown(remaining)
      : sleepTimer.kind === "chapter"
        ? t("chapterWord")
        : sleepTimer.kind === "episode"
          ? t("episodeWord")
          : t("sleep");

  return (
    <Menu
      label={label}
      className="menu-inline"
      trigger={
        <>
          <Icon name="moon" size={18} />
          <span>{triggerLabel}</span>
        </>
      }
    >
      {(close) =>
        SLEEP_TIMER_OPTIONS.map((option) => (
          <button
            key={option}
            type="button"
            role="menuitemradio"
            aria-checked={isSelected(option, sleepTimer)}
            className="menu-item"
            onClick={() => {
              setSleepTimerOption(option);
              close();
            }}
          >
            <span className="menu-check" aria-hidden="true">
              {isSelected(option, sleepTimer) ? <Icon name="check" size={16} /> : null}
            </span>
            <span>{sleepTimerOptionLabel(option)}</span>
          </button>
        ))
      }
    </Menu>
  );
}
