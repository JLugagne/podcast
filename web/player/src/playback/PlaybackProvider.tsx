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
import { getCatalog, getEpisode, resolveAudioURL } from "../api/client";
import type { Catalog, ChapterDTO, Episode, PlaybackSeries } from "../api/types";
import { queryKeys } from "../hooks/keys";
import {
  chapterEnd,
  chapterIndexAt,
  nextChapterTarget,
  previousChapterTarget,
  sortChapters,
} from "../lib/chapters";
import { downloadedObjectURL } from "../lib/downloads";
import {
  clearMediaSessionMetadata,
  setMediaSessionHandlers,
  setMediaSessionMetadata,
  setMediaSessionPositionState,
} from "../lib/mediaSession";
import { getProgress, markPlayed, markUnplayed, recordProgress } from "../lib/progress";
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
  restart: () => void;
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
  const [chapters, setChapters] = useState<ChapterDTO[]>([]);
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
  const objectUrlRef = useRef<string | null>(null);
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

  const announceChapter = useCallback((index: number, list: ChapterDTO[]) => {
    const chapter = list[index];
    setAnnouncement(chapter ? `Now playing chapter ${index + 1}: ${chapter.title}` : "");
  }, []);

  const saveProgress = useCallback((value: number, options: { force?: boolean } = {}) => {
    const current = episodeRef.current;
    if (!current || !Number.isFinite(value) || value <= 0) return;
    const now = Date.now();
    if (!options.force && !throttleRef.current.shouldSave(now)) return;
    throttleRef.current.markSaved(now);
    recordProgress(current.id, value, durationRef.current);
  }, []);

  const pauseForSleepTimer = useCallback(() => {
    engine.pause();
    setPlayingState(false);
  }, [engine, setPlayingState]);

  const loadChapters = useCallback(
    async (nextEpisode: Episode, knownDuration: number) => {
      try {
        const detail = await getEpisode(nextEpisode.id);
        if (episodeRef.current?.id !== nextEpisode.id) return;
        const ordered = sortChapters(detail.chapters);
        chaptersRef.current = ordered;
        setChapters(ordered);
        lastChapterIndexRef.current = chapterIndexAt(ordered, positionRef.current);
        announceChapter(lastChapterIndexRef.current, ordered);
        if (knownDuration <= 0 && detail.duration_seconds > 0) {
          setDurationState(detail.duration_seconds);
        }
      } catch {
        // Chapters are optional: playback continues without them.
      }
    },
    [announceChapter, setDurationState],
  );

  const start = useCallback(
    (nextEpisode: Episode, nextSeries: PlaybackSeries, positionSeconds = 0) => {
      if (!nextEpisode.audio_url) {
        setError("This episode has no audio file.");
        return;
      }
      const startAt = Math.max(0, positionSeconds);
      const knownDuration = Math.max(0, nextEpisode.duration_seconds);

      episodeRef.current = nextEpisode;
      seriesRef.current = toPlaybackSeries(nextSeries);
      chaptersRef.current = [];
      durationRef.current = knownDuration;
      lastChapterIndexRef.current = -1;
      throttleRef.current = new ProgressThrottle(PROGRESS_SAVE_INTERVAL_MS, Date.now());

      setEpisode(nextEpisode);
      setSeries(toPlaybackSeries(nextSeries));
      setChapters([]);
      updatePosition(startAt);
      setDurationState(knownDuration);
      setError(null);
      if (sleepTimerRef.current.kind === "chapter") setSleepTimer({ kind: "off" });

      setPlayingState(true);
      void downloadedObjectURL(nextEpisode.id)
        .catch(() => null)
        .then((offlineUrl) => {
          // A newer start() call may have replaced this episode while resolving.
          if (episodeRef.current?.id !== nextEpisode.id) return;
          if (offlineUrl) {
            if (objectUrlRef.current) URL.revokeObjectURL(objectUrlRef.current);
            objectUrlRef.current = offlineUrl;
          }
          engine.load(offlineUrl ?? resolveAudioURL(nextEpisode.audio_url), startAt);
          engine.setPlaybackRate(rateRef.current);
          return engine.play();
        })
        .catch(() => {
          setPlayingState(false);
          setError("The audio could not be played.");
        });
      void loadChapters(nextEpisode, knownDuration);
    },
    [engine, updatePosition, setDurationState, setSleepTimer, setPlayingState, loadChapters],
  );

  const seekTo = useCallback(
    (seconds: number) => {
      if (!episodeRef.current) return;
      const maximum = durationRef.current;
      const value = maximum > 0 ? Math.min(Math.max(0, seconds), maximum) : Math.max(0, seconds);
      engine.seek(value);
      updatePosition(value);
      const list = chaptersRef.current;
      const index = chapterIndexAt(list, value);
      if (index !== lastChapterIndexRef.current) {
        lastChapterIndexRef.current = index;
        announceChapter(index, list);
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

  /** Plays the current episode again from 0:00 and forgets that it was played. */
  const restart = useCallback(() => {
    const current = episodeRef.current;
    if (!current) return;
    engine.seek(0);
    updatePosition(0);
    const list = chaptersRef.current;
    lastChapterIndexRef.current = chapterIndexAt(list, 0);
    announceChapter(lastChapterIndexRef.current, list);
    markUnplayed(current.id);
    void queryClient.invalidateQueries({ queryKey: queryKeys.listeningAll });
    if (playingRef.current) return;
    setPlayingState(true);
    setError(null);
    void engine.play().catch(() => {
      setPlayingState(false);
      setError("The audio could not be played.");
    });
  }, [engine, updatePosition, announceChapter, queryClient, setPlayingState]);

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
      let catalog: Catalog;
      try {
        catalog = await queryClient.fetchQuery({
          queryKey: queryKeys.catalog,
          queryFn: getCatalog,
          staleTime: Number.POSITIVE_INFINITY,
        });
      } catch {
        return;
      }

      const seriesData = catalog.series.find((item) => item.slug === current.series_slug);
      if (!seriesData) return;
      const index = seriesData.episodes.findIndex((candidate) => candidate.id === current.id);
      const next = index >= 0 ? seriesData.episodes[index + 1] : undefined;
      if (!next) return;

      const row = getProgress(next.id);
      const resumeSeconds =
        row && !row.completed && row.position_seconds > 0 ? row.position_seconds : 0;
      start(
        next,
        {
          slug: seriesData.slug,
          title: seriesData.title,
          accent_color: seriesData.accent_color,
          cover_url: seriesData.cover_url,
        },
        resumeSeconds,
      );
    },
    [queryClient, start],
  );

  const handleEnded = useCallback(() => {
    const current = episodeRef.current;
    setPlayingState(false);
    if (!current) return;
    markPlayed(current.id, durationRef.current);
    const timer = sleepTimerRef.current;
    void queryClient.invalidateQueries({ queryKey: queryKeys.listeningAll });
    setSleepTimer({ kind: "off" });
    if (shouldStopAtEpisodeEnd(timer)) return;
    if (!autoPlayNextRef.current) return;
    void playNextEpisode(current);
  }, [playNextEpisode, queryClient, setPlayingState, setSleepTimer]);

  const handleEngineEvent = useCallback(
    (event: MediaEngineEvent) => {
      switch (event.type) {
        case "time": {
          updatePosition(event.currentTime);
          if (event.duration > 0) setDurationState(event.duration);
          const list = chaptersRef.current;
          const index = chapterIndexAt(list, event.currentTime);
          const chapterChanged = index !== lastChapterIndexRef.current;
          if (chapterChanged) {
            lastChapterIndexRef.current = index;
            announceChapter(index, list);
          }
          if (playingRef.current) saveProgress(event.currentTime, { force: chapterChanged });
          const timer = sleepTimerRef.current;
          if (timer.kind === "chapter") {
            const end = chapterEnd(list, timer.chapterIndex, event.duration || durationRef.current);
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
      if (current) recordProgress(current.id, positionRef.current, durationRef.current);
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
      restart,
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
      restart,
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
