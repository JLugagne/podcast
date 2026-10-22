import { useQueryClient } from "@tanstack/react-query";
import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { api, saveProgressOnUnload } from "../api/client";
import type { ChapterDTO, Episode, PlaybackSeries } from "../api/types";
import { queryKeys } from "../hooks/keys";
import {
  chapterEnd,
  chapterIndexAt,
  nextChapterTarget,
  previousChapterTarget,
  sortChapters,
} from "../lib/chapters";
import {
  clearMediaSessionMetadata,
  setMediaSessionHandlers,
  setMediaSessionMetadata,
  setMediaSessionPositionState,
} from "../lib/mediaSession";
import { PROGRESS_SAVE_INTERVAL_MS, ProgressThrottle } from "../lib/progressThrottle";
import {
  type SleepTimer,
  type SleepTimerOption,
  shouldPauseForChapterTimer,
  shouldStopAtEpisodeEnd,
  sleepTimerFromOption,
  sleepTimerRemainingSeconds,
} from "../lib/sleepTimer";
import {
  AUTO_PLAY_NEXT_KEY,
  PLAYBACK_RATE_KEY,
  readBoolean,
  readNumber,
  writeBoolean,
  writeNumber,
} from "../lib/storage";
import { HtmlAudioEngine, type MediaEngine, type MediaEngineEvent } from "./engine";
import { normalizeRate, type PlaybackRate } from "./rates";

export interface PlaybackContextValue {
  episode: Episode | null;
  series: PlaybackSeries | null;
  chapters: ChapterDTO[];
  chapterIndex: number;
  currentChapter: ChapterDTO | null;
  position: number;
  duration: number;
  playing: boolean;
  rate: PlaybackRate;
  autoPlayNext: boolean;
  sleepTimer: SleepTimer;
  sleepRemaining: number | null;
  error: string | null;
  start: (episode: Episode, series: PlaybackSeries, positionSeconds?: number) => void;
  toggle: () => void;
  seekTo: (seconds: number) => void;
  skip: (deltaSeconds: number) => void;
  nextChapter: () => void;
  previousChapter: () => void;
  setRate: (rate: number) => void;
  setAutoPlayNext: (value: boolean) => void;
  setSleepTimerOption: (option: SleepTimerOption) => void;
}

const PlaybackContext = createContext<PlaybackContextValue | null>(null);

export function usePlayback(): PlaybackContextValue {
  const context = useContext(PlaybackContext);
  if (!context) throw new Error("usePlayback must be used inside a PlaybackProvider.");
  return context;
}

function toPlaybackSeries(series: PlaybackSeries): PlaybackSeries {
  return {
    id: series.id,
    slug: series.slug,
    title: series.title,
    accent_color: series.accent_color,
    cover_url: series.cover_url,
  };
}

export interface PlaybackProviderProps {
  children: ReactNode;
  /** Injected engine, used by tests; production creates one HTMLAudioEngine. */
  engine?: MediaEngine;
}

export function PlaybackProvider({ children, engine: injectedEngine }: PlaybackProviderProps) {
  const queryClient = useQueryClient();
  const [engine] = useState<MediaEngine>(() => injectedEngine ?? new HtmlAudioEngine());

  const [episode, setEpisode] = useState<Episode | null>(null);
  const [series, setSeries] = useState<PlaybackSeries | null>(null);
  const [position, setPosition] = useState(0);
  const [duration, setDuration] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [rate, setRateState] = useState<PlaybackRate>(() =>
    normalizeRate(readNumber(PLAYBACK_RATE_KEY, 1)),
  );
  const [autoPlayNext, setAutoPlayNextState] = useState(() =>
    readBoolean(AUTO_PLAY_NEXT_KEY, true),
  );
  const [sleepTimer, setSleepTimerState] = useState<SleepTimer>({ kind: "off" });
  const [sleepRemaining, setSleepRemaining] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [announcement, setAnnouncement] = useState("");

  const episodeRef = useRef<Episode | null>(null);
  const seriesRef = useRef<PlaybackSeries | null>(null);
  const chaptersRef = useRef<ChapterDTO[]>([]);
  const positionRef = useRef(0);
  const durationRef = useRef(0);
  const playingRef = useRef(false);
  const rateRef = useRef<PlaybackRate>(rate);
  const autoPlayNextRef = useRef(autoPlayNext);
  const sleepTimerRef = useRef<SleepTimer>({ kind: "off" });
  const throttleRef = useRef(new ProgressThrottle(PROGRESS_SAVE_INTERVAL_MS, 0));
  const lastChapterIndexRef = useRef(-1);

  const updatePosition = useCallback((value: number) => {
    positionRef.current = value;
    setPosition(value);
  }, []);

  const setDurationState = useCallback((value: number) => {
    durationRef.current = value;
    setDuration(value);
  }, []);

  const setPlayingState = useCallback((value: boolean) => {
    playingRef.current = value;
    setPlaying(value);
  }, []);

  const setSleepTimer = useCallback((timer: SleepTimer) => {
    sleepTimerRef.current = timer;
    setSleepTimerState(timer);
  }, []);

  const announceChapter = useCallback((index: number, chapters: ChapterDTO[]) => {
    const chapter = chapters[index];
    setAnnouncement(chapter ? `Now playing chapter ${index + 1}: ${chapter.title}` : "");
  }, []);

  const saveProgress = useCallback((value: number, options: { force?: boolean } = {}) => {
    const current = episodeRef.current;
    if (!current || !Number.isFinite(value) || value <= 0) return;
    const now = Date.now();
    if (!options.force && !throttleRef.current.shouldSave(now)) return;
    throttleRef.current.markSaved(now);
    void api.recordProgress(current.id, value).catch(() => undefined);
  }, []);

  const pauseForSleepTimer = useCallback(() => {
    engine.pause();
    setPlayingState(false);
  }, [engine, setPlayingState]);

  const start = useCallback(
    (nextEpisode: Episode, nextSeries: PlaybackSeries, positionSeconds = 0) => {
      if (!nextEpisode.audio) {
        setError("This episode has no audio file.");
        return;
      }
      const chapters = sortChapters(nextEpisode.chapters);
      const startAt = Math.max(0, positionSeconds);
      const knownDuration = Math.max(0, nextEpisode.audio.duration_seconds);

      episodeRef.current = nextEpisode;
      seriesRef.current = toPlaybackSeries(nextSeries);
      chaptersRef.current = chapters;
      durationRef.current = knownDuration;
      lastChapterIndexRef.current = chapterIndexAt(chapters, startAt);
      throttleRef.current = new ProgressThrottle(PROGRESS_SAVE_INTERVAL_MS, Date.now());

      setEpisode(nextEpisode);
      setSeries(toPlaybackSeries(nextSeries));
      updatePosition(startAt);
      setDurationState(knownDuration);
      setError(null);
      if (sleepTimerRef.current.kind === "chapter") setSleepTimer({ kind: "off" });

      engine.load(nextEpisode.audio.url, startAt);
      engine.setPlaybackRate(rateRef.current);
      setPlayingState(true);
      announceChapter(lastChapterIndexRef.current, chapters);
      void engine.play().catch(() => {
        setPlayingState(false);
        setError("The audio could not be played.");
      });
    },
    [engine, updatePosition, setDurationState, setSleepTimer, announceChapter, setPlayingState],
  );

  const seekTo = useCallback(
    (seconds: number) => {
      if (!episodeRef.current) return;
      const maximum = durationRef.current;
      const value = maximum > 0 ? Math.min(Math.max(0, seconds), maximum) : Math.max(0, seconds);
      engine.seek(value);
      updatePosition(value);
      const chapters = chaptersRef.current;
      const index = chapterIndexAt(chapters, value);
      if (index !== lastChapterIndexRef.current) {
        lastChapterIndexRef.current = index;
        announceChapter(index, chapters);
      }
      saveProgress(value, { force: true });
    },
    [engine, updatePosition, announceChapter, saveProgress],
  );

  const skip = useCallback(
    (deltaSeconds: number) => {
      seekTo(positionRef.current + deltaSeconds);
    },
    [seekTo],
  );

  const toggle = useCallback(() => {
    if (!episodeRef.current) return;
    if (playingRef.current) {
      engine.pause();
      setPlayingState(false);
      saveProgress(positionRef.current, { force: true });
      return;
    }
    setPlayingState(true);
    setError(null);
    void engine.play().catch(() => {
      setPlayingState(false);
      setError("The audio could not be played.");
    });
  }, [engine, setPlayingState, saveProgress]);

  const nextChapter = useCallback(() => {
    const target = nextChapterTarget(chaptersRef.current, positionRef.current);
    if (target) seekTo(target.seconds);
  }, [seekTo]);

  const previousChapter = useCallback(() => {
    const target = previousChapterTarget(chaptersRef.current, positionRef.current);
    if (target) seekTo(target.seconds);
  }, [seekTo]);

  const playNextEpisode = useCallback(
    async (current: Episode) => {
      let episodes: Episode[];
      try {
        const response = await queryClient.fetchQuery({
          queryKey: queryKeys.publishedEpisodes(current.series_id),
          queryFn: () => api.listEpisodes(current.series_id, "published"),
          staleTime: 0,
        });
        episodes = response.episodes;
      } catch {
        return;
      }

      const index = episodes.findIndex((candidate) => candidate.id === current.id);
      const next = index >= 0 ? episodes[index + 1] : undefined;
      if (!next) return;

      let resumeSeconds = 0;
      try {
        const response = await queryClient.fetchQuery({
          queryKey: queryKeys.progress(),
          queryFn: () => api.listProgress(),
          staleTime: 0,
        });
        const row = response.progress.find((item) => item.episode_id === next.id);
        if (row && !row.completed && row.position_seconds > 0) {
          resumeSeconds = row.position_seconds;
        }
      } catch {
        // Start the next episode from the beginning when progress is unavailable.
      }

      let nextSeries = seriesRef.current;
      if (!nextSeries || nextSeries.id !== next.series_id) {
        try {
          nextSeries = toPlaybackSeries(await api.getSeries(next.series_id));
        } catch {
          return;
        }
      }
      start(next, nextSeries, resumeSeconds);
    },
    [queryClient, start],
  );

  const handleEnded = useCallback(() => {
    const current = episodeRef.current;
    setPlayingState(false);
    if (!current) return;
    const timer = sleepTimerRef.current;
    void (async () => {
      try {
        await api.markPlayed(current.id);
      } catch {
        // Keep the local state consistent even when the save fails.
      }
      void queryClient.invalidateQueries({ queryKey: queryKeys.listeningAll });
      setSleepTimer({ kind: "off" });
      if (shouldStopAtEpisodeEnd(timer)) return;
      if (!autoPlayNextRef.current) return;
      await playNextEpisode(current);
    })();
  }, [playNextEpisode, queryClient, setPlayingState, setSleepTimer]);

  const handleEngineEvent = useCallback(
    (event: MediaEngineEvent) => {
      switch (event.type) {
        case "time": {
          updatePosition(event.currentTime);
          if (event.duration > 0) setDurationState(event.duration);
          const chapters = chaptersRef.current;
          const index = chapterIndexAt(chapters, event.currentTime);
          const chapterChanged = index !== lastChapterIndexRef.current;
          if (chapterChanged) {
            lastChapterIndexRef.current = index;
            announceChapter(index, chapters);
          }
          if (playingRef.current) saveProgress(event.currentTime, { force: chapterChanged });
          const timer = sleepTimerRef.current;
          if (timer.kind === "chapter") {
            const end = chapterEnd(
              chapters,
              timer.chapterIndex,
              event.duration || durationRef.current,
            );
            if (shouldPauseForChapterTimer(timer, event.currentTime, end)) {
              setSleepTimer({ kind: "off" });
              pauseForSleepTimer();
            }
          }
          break;
        }
        case "duration":
          if (event.duration > 0) setDurationState(event.duration);
          break;
        case "play":
          setPlayingState(true);
          break;
        case "pause":
          setPlayingState(false);
          saveProgress(positionRef.current, { force: true });
          break;
        case "ended":
          handleEnded();
          break;
        case "error":
          setError(event.message);
          setPlayingState(false);
          break;
      }
    },
    [
      updatePosition,
      setDurationState,
      announceChapter,
      saveProgress,
      setSleepTimer,
      pauseForSleepTimer,
      setPlayingState,
      handleEnded,
    ],
  );

  useEffect(() => engine.subscribe(handleEngineEvent), [engine, handleEngineEvent]);

  useEffect(() => {
    const handlePageHide = () => {
      const current = episodeRef.current;
      if (current) saveProgressOnUnload(current.id, positionRef.current);
    };
    window.addEventListener("pagehide", handlePageHide);
    return () => window.removeEventListener("pagehide", handlePageHide);
  }, []);

  useEffect(() => {
    if (sleepTimer.kind !== "minutes") {
      setSleepRemaining(null);
      return;
    }
    const tick = () => {
      const remaining = sleepTimerRemainingSeconds(sleepTimer, Date.now());
      setSleepRemaining(remaining);
      if (remaining !== null && remaining <= 0) {
        setSleepTimer({ kind: "off" });
        pauseForSleepTimer();
      }
    };
    tick();
    const interval = window.setInterval(tick, 1000);
    return () => window.clearInterval(interval);
  }, [sleepTimer, pauseForSleepTimer, setSleepTimer]);

  const chapters = useMemo(() => sortChapters(episode?.chapters ?? []), [episode]);
  const chapterIndex = chapters.length > 0 ? chapterIndexAt(chapters, position) : -1;
  const currentChapter = chapterIndex >= 0 ? chapters[chapterIndex] : null;
  const currentChapterTitle = currentChapter?.title ?? null;

  useEffect(() => {
    if (!episode || !series) {
      clearMediaSessionMetadata();
      return;
    }
    setMediaSessionMetadata({
      title: episode.title,
      artist: series.title,
      album: currentChapterTitle ?? series.title,
      artworkUrl: series.cover_url,
    });
  }, [episode, series, currentChapterTitle]);

  useEffect(() => {
    setMediaSessionHandlers({
      play: () => {
        if (!playingRef.current) toggle();
      },
      pause: () => {
        if (playingRef.current) toggle();
      },
      seekbackward: (details) => skip(-(details.seekOffset ?? 15)),
      seekforward: (details) => skip(details.seekOffset ?? 30),
      previoustrack: () => previousChapter(),
      nexttrack: () => nextChapter(),
      seekto: (details) => {
        if (typeof details.seekTime === "number") seekTo(details.seekTime);
      },
    });
    return () => {
      setMediaSessionHandlers({
        play: null,
        pause: null,
        seekbackward: null,
        seekforward: null,
        previoustrack: null,
        nexttrack: null,
        seekto: null,
      });
    };
  }, [toggle, skip, previousChapter, nextChapter, seekTo]);

  useEffect(() => {
    if (!episode) return;
    setMediaSessionPositionState(duration, position, rate);
  }, [episode, duration, position, rate]);

  const setRate = useCallback(
    (value: number) => {
      const next = normalizeRate(value);
      rateRef.current = next;
      setRateState(next);
      engine.setPlaybackRate(next);
      writeNumber(PLAYBACK_RATE_KEY, next);
    },
    [engine],
  );

  const setAutoPlayNext = useCallback((value: boolean) => {
    autoPlayNextRef.current = value;
    setAutoPlayNextState(value);
    writeBoolean(AUTO_PLAY_NEXT_KEY, value);
  }, []);

  const setSleepTimerOption = useCallback(
    (option: SleepTimerOption) => {
      const timer = sleepTimerFromOption(option, {
        nowMs: Date.now(),
        chapterIndex: chapterIndexAt(chaptersRef.current, positionRef.current),
      });
      setSleepTimer(timer);
    },
    [setSleepTimer],
  );

  const value = useMemo<PlaybackContextValue>(
    () => ({
      episode,
      series,
      chapters,
      chapterIndex,
      currentChapter,
      position,
      duration,
      playing,
      rate,
      autoPlayNext,
      sleepTimer,
      sleepRemaining,
      error,
      start,
      toggle,
      seekTo,
      skip,
      nextChapter,
      previousChapter,
      setRate,
      setAutoPlayNext,
      setSleepTimerOption,
    }),
    [
      episode,
      series,
      chapters,
      chapterIndex,
      currentChapter,
      position,
      duration,
      playing,
      rate,
      autoPlayNext,
      sleepTimer,
      sleepRemaining,
      error,
      start,
      toggle,
      seekTo,
      skip,
      nextChapter,
      previousChapter,
      setRate,
      setAutoPlayNext,
      setSleepTimerOption,
    ],
  );

  return (
    <PlaybackContext.Provider value={value}>
      {children}
      <div className="sr-only" role="status" aria-live="polite">
        {announcement}
      </div>
    </PlaybackContext.Provider>
  );
}
