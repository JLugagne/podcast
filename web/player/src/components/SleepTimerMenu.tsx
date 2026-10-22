import {
  formatCountdown,
  SLEEP_TIMER_OPTIONS,
  type SleepTimer,
  type SleepTimerOption,
  sleepTimerLabel,
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

function triggerText(timer: SleepTimer, remaining: number) {
  if (timer.kind === "minutes") return formatCountdown(remaining);
  if (timer.kind === "chapter") return "Chapter";
  if (timer.kind === "episode") return "Episode";
  return "Sleep";
}

export function SleepTimerMenu() {
  const { sleepTimer, sleepRemaining, setSleepTimerOption } = usePlayback();
  const label = sleepTimerLabel(sleepTimer, sleepRemaining);
  const remaining = sleepTimer.kind === "minutes" ? (sleepRemaining ?? sleepTimer.minutes * 60) : 0;

  return (
    <Menu
      label={label}
      className="menu-inline"
      trigger={
        <>
          <Icon name="moon" size={18} />
          <span>{triggerText(sleepTimer, remaining)}</span>
        </>
      }
    >
      {(close) =>
        SLEEP_TIMER_OPTIONS.map((option) => (
          <button
            key={option.value}
            type="button"
            role="menuitemradio"
            aria-checked={isSelected(option.value, sleepTimer)}
            className="menu-item"
            onClick={() => {
              setSleepTimerOption(option.value);
              close();
            }}
          >
            <span className="menu-check" aria-hidden="true">
              {isSelected(option.value, sleepTimer) ? <Icon name="check" size={16} /> : null}
            </span>
            <span>{option.label}</span>
          </button>
        ))
      }
    </Menu>
  );
}
