import type { FormEvent, RefObject } from "react";
import { useMemo, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { parseChaptersDocument, serializeChapters } from "../api/chapters";
import { errorMessage, isAbortError, isVersionConflict } from "../api/client";
import { API_BASE } from "../api/errors";
import type { Episode, UploadProgress } from "../api/types";
import { AudioPlayer } from "../components/AudioPlayer";
import { Button } from "../components/Button";
import { ChapterTable } from "../components/ChapterTable";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { ErrorBanner, InfoBanner } from "../components/ErrorBanner";
import { TextAreaField, TextField } from "../components/Field";
import { FileDrop } from "../components/FileDrop";
import { LoadingBlock } from "../components/LoadingBlock";
import { ProgressBar } from "../components/ProgressBar";
import { StatusBadge } from "../components/StatusBadge";
import {
  useDeleteEpisode,
  useEpisode,
  usePublishEpisode,
  useSetChapters,
  useUnpublishEpisode,
  useUpdateEpisode,
  useUploadAudio,
} from "../hooks/episodes";
import { useSeries } from "../hooks/series";
import { formatBytes, formatDateTime, formatDuration, formatTimecode } from "../lib/format";
import { isAudioFile, readAudioDuration } from "../lib/media";
import { withCacheBust } from "../lib/urls";

const MAX_CHAPTERS_FILE_BYTES = 2 * 1024 * 1024;

export function EpisodeEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const episodeQuery = useEpisode(id);
  const seriesQuery = useSeries(episodeQuery.data?.series_id);
  const deleteEpisode = useDeleteEpisode();
  const audioRef = useRef<HTMLAudioElement>(null);

  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  if (episodeQuery.isPending) {
    return (
      <div className="page">
        <LoadingBlock label="Loading episode…" />
      </div>
    );
  }

  if (episodeQuery.isError || !episodeQuery.data) {
    return (
      <div className="page">
        <ErrorBanner
          title="Could not load this episode"
          message={
            episodeQuery.isError ? errorMessage(episodeQuery.error) : "The episode was not found."
          }
        />
        <p className="mt-4">
          <Link to="/">Back to series</Link>
        </p>
      </div>
    );
  }

  const episode = episodeQuery.data;
  const series = seriesQuery.data;

  const seek = (seconds: number) => {
    const audio = audioRef.current;
    if (!audio) return;
    audio.currentTime = seconds;
    void audio.play().catch(() => {
      // Playback may be blocked until the user interacts with the player.
    });
  };

  async function handleDelete() {
    setDeleteError(null);
    try {
      await deleteEpisode.mutateAsync({ id: episode.id, seriesId: episode.series_id });
      navigate(series ? `/series/${series.id}` : "/");
    } catch (error) {
      setDeleteError(errorMessage(error));
    }
  }

  return (
    <div className="page">
      <nav className="breadcrumbs" aria-label="Breadcrumb">
        <Link to="/">Series</Link>
        <span aria-hidden="true">/</span>
        {series ? (
          <>
            <Link to={`/series/${series.id}`}>{series.title}</Link>
            <span aria-hidden="true">/</span>
          </>
        ) : null}
        <span aria-current="page">Episode {episode.number}</span>
      </nav>

      <header className="page-header">
        <div>
          <div className="row">
            <StatusBadge status={episode.status} />
            <span className="muted text-small">Episode {episode.number}</span>
          </div>
          <h1 className="page-title mt-2">{episode.title}</h1>
        </div>
        <div className="row">
          <Button
            variant="danger"
            onClick={() => {
              setDeleteError(null);
              setDeleteOpen(true);
            }}
          >
            Delete episode
          </Button>
        </div>
      </header>

      <div className="editor-layout">
        <div className="stack">
          <MetadataPanel key={`details-${episode.id}`} episode={episode} />
          <AudioPanel key={`audio-${episode.id}`} episode={episode} audioRef={audioRef} />
          <ChaptersPanel
            key={`chapters-${episode.id}`}
            episode={episode}
            onSeek={episode.audio ? seek : undefined}
            hasAudio={Boolean(episode.audio)}
          />
        </div>
        <div className="side-panel">
          <PublishPanel episode={episode} />
        </div>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        title={`Delete “${episode.title}”?`}
        description="The episode, its audio and its chapters will be permanently deleted. This action cannot be undone."
        confirmLabel="Delete episode"
        busy={deleteEpisode.isPending}
        error={deleteError}
        onCancel={() => setDeleteOpen(false)}
        onConfirm={() => void handleDelete()}
      />
    </div>
  );
}

function MetadataPanel({ episode }: { episode: Episode }) {
  const updateEpisode = useUpdateEpisode();
  const [number, setNumber] = useState(String(episode.number));
  const [title, setTitle] = useState(episode.title);
  const [description, setDescription] = useState(episode.description);
  const [submitted, setSubmitted] = useState(false);
  const [conflict, setConflict] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const parsedNumber = Number.parseInt(number, 10);
  const numberError =
    number.trim().length === 0 || !Number.isInteger(parsedNumber) || parsedNumber < 1
      ? "The episode number must be a positive integer."
      : undefined;
  const titleError = !title.trim()
    ? "A title is required."
    : title.trim().length > 200
      ? "The title must be at most 200 characters."
      : undefined;
  const descriptionError =
    description.length > 10000 ? "The description must be at most 10000 characters." : undefined;
  const invalid = Boolean(numberError || titleError || descriptionError);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitted(true);
    setConflict(false);
    setFormError(null);
    setSaved(false);
    if (invalid) return;

    try {
      await updateEpisode.mutateAsync({
        id: episode.id,
        input: {
          number: parsedNumber,
          title: title.trim(),
          description,
          version: episode.version,
        },
      });
      setSaved(true);
    } catch (error) {
      if (isVersionConflict(error)) {
        setConflict(true);
      } else {
        setFormError(errorMessage(error));
      }
    }
  }

  return (
    <section className="card panel" aria-labelledby="episode-details-heading">
      <div className="panel-header">
        <h2 className="panel-title" id="episode-details-heading">
          Details
        </h2>
        <span className="muted text-small">Version {episode.version}</span>
      </div>

      <form className="stack" onSubmit={handleSubmit} noValidate>
        {conflict ? (
          <div className="stack">
            <ErrorBanner
              title="This episode changed elsewhere"
              message="Someone or another tab updated this episode. Reload the page to get the latest version before saving again."
            />
            <div>
              <Button variant="secondary" onClick={() => window.location.reload()}>
                Reload the page
              </Button>
            </div>
          </div>
        ) : null}
        {formError ? <ErrorBanner message={formError} /> : null}
        {saved ? (
          <div className="banner banner-success" role="status">
            <span className="banner-icon" aria-hidden="true">
              ✓
            </span>
            <span>Episode details saved.</span>
          </div>
        ) : null}

        <div className="grid-two">
          <TextField
            id="episode-number"
            label="Number"
            type="number"
            min={1}
            step={1}
            value={number}
            onChange={(event) => {
              setNumber(event.target.value);
              setSaved(false);
            }}
            required
            error={submitted ? numberError : undefined}
          />
          <TextField
            id="episode-title"
            label="Title"
            value={title}
            onChange={(event) => {
              setTitle(event.target.value);
              setSaved(false);
            }}
            required
            maxLength={200}
            error={submitted ? titleError : undefined}
          />
        </div>
        <TextAreaField
          id="episode-description"
          label="Description"
          value={description}
          onChange={(event) => {
            setDescription(event.target.value);
            setSaved(false);
          }}
          maxLength={10000}
          hint={`${description.length}/10000 characters`}
          error={submitted ? descriptionError : undefined}
        />
        <div className="row">
          <Button type="submit" variant="primary" disabled={updateEpisode.isPending}>
            {updateEpisode.isPending ? "Saving…" : "Save details"}
          </Button>
        </div>
      </form>
    </section>
  );
}

interface AudioPanelProps {
  episode: Episode;
  audioRef: RefObject<HTMLAudioElement | null>;
}

function AudioPanel({ episode, audioRef }: AudioPanelProps) {
  const uploadAudio = useUploadAudio();
  const [phase, setPhase] = useState<"idle" | "reading" | "uploading">("idle");
  const [progress, setProgress] = useState<UploadProgress | null>(null);
  const [pendingName, setPendingName] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [note, setNote] = useState<string | null>(null);
  const controllerRef = useRef<AbortController | null>(null);

  const audio = episode.audio;
  const audioSrc = audio ? (withCacheBust(audio.url, episode.version) ?? audio.url) : null;

  async function handleFile(file: File) {
    setError(null);
    setNote(null);
    if (!isAudioFile(file)) {
      setError("Choose an audio file (MP3, M4A, AAC, OGG, Opus, WAV, WebM or FLAC).");
      return;
    }

    setPendingName(file.name);
    setPhase("reading");
    const durationSeconds = await readAudioDuration(file);
    if (durationSeconds === 0) {
      setNote("The browser could not read the duration; it will be stored as unknown.");
    }

    const controller = new AbortController();
    controllerRef.current = controller;
    setProgress({ percent: 0, loaded: 0, total: file.size });
    setPhase("uploading");

    try {
      await uploadAudio.mutateAsync({
        id: episode.id,
        file,
        durationSeconds,
        signal: controller.signal,
        onProgress: setProgress,
      });
      setPendingName(null);
    } catch (uploadError) {
      if (isAbortError(uploadError)) {
        setNote("Upload cancelled.");
        setPendingName(null);
      } else {
        setError(errorMessage(uploadError));
      }
    } finally {
      controllerRef.current = null;
      setProgress(null);
      setPhase("idle");
    }
  }

  const working = phase !== "idle";

  return (
    <section className="card panel" aria-labelledby="episode-audio-heading">
      <div className="panel-header">
        <h2 className="panel-title" id="episode-audio-heading">
          Audio
        </h2>
        {audio ? <span className="badge badge-published">Uploaded</span> : null}
      </div>

      {audio && audioSrc ? (
        <div className="stack">
          <div className="file-meta">
            <span>
              <strong>{audio.original_name}</strong>
            </span>
            <span>{formatBytes(audio.size_bytes)}</span>
            <span>
              {audio.duration_seconds > 0
                ? formatDuration(audio.duration_seconds)
                : "Duration unknown"}
            </span>
            <span className="mono">{audio.content_type}</span>
          </div>
          <AudioPlayer src={audioSrc} audioRef={audioRef} />
        </div>
      ) : (
        <p className="muted">
          No audio uploaded yet. An episode needs audio before it can be published.
        </p>
      )}

      <div className="stack mt-4">
        {error ? <ErrorBanner message={error} /> : null}
        {note ? <InfoBanner message={note} /> : null}

        <FileDrop
          id="episode-audio-file"
          label={working ? "Uploading audio…" : audio ? "Replace audio file" : "Upload audio file"}
          hint="Drag and drop an audio file here, or choose one (MP3, M4A, AAC, OGG, Opus, WAV, WebM, FLAC)."
          accept="audio/*"
          disabled={working}
          onFile={(file) => void handleFile(file)}
        />

        <div aria-live="polite" className="stack">
          {phase === "reading" ? (
            <p className="muted text-small">Reading the audio duration in your browser…</p>
          ) : null}
          {phase === "uploading" && progress ? (
            <div className="stack">
              <div className="upload-status">
                <span>Uploading {pendingName}</span>
                <span>{progress.percent}%</span>
              </div>
              <ProgressBar value={progress.percent} label="Audio upload progress" />
              <div>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => controllerRef.current?.abort()}
                >
                  Cancel upload
                </Button>
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </section>
  );
}

interface ChaptersPanelProps {
  episode: Episode;
  hasAudio: boolean;
  onSeek?: (seconds: number) => void;
}

function ChaptersPanel({ episode, hasAudio, onSeek }: ChaptersPanelProps) {
  const setChapters = useSetChapters();
  const [text, setText] = useState(() =>
    episode.chapters.length > 0 ? serializeChapters(episode.chapters) : "",
  );
  const [sourceName, setSourceName] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const parsed = useMemo(
    () => (text.trim().length > 0 ? parseChaptersDocument(text) : null),
    [text],
  );

  function updateText(value: string) {
    setText(value);
    setSaved(false);
    setSaveError(null);
    setSourceName(null);
  }

  async function handleFile(file: File) {
    setSaveError(null);
    setSaved(false);
    const looksJson = file.type === "application/json" || file.name.toLowerCase().endsWith(".json");
    if (!looksJson) {
      setSaveError("Choose a .json chapters document.");
      return;
    }
    if (file.size > MAX_CHAPTERS_FILE_BYTES) {
      setSaveError("That file is too large for a chapters document.");
      return;
    }
    try {
      const content = await file.text();
      setText(content);
      setSourceName(file.name);
    } catch {
      setSaveError("Could not read that file.");
    }
  }

  async function handleSave() {
    if (!parsed?.ok) return;
    setSaveError(null);
    setSaved(false);
    try {
      await setChapters.mutateAsync({ id: episode.id, raw: text });
      setSaved(true);
    } catch (error) {
      setSaveError(errorMessage(error));
    }
  }

  const chapterCount = parsed?.ok ? parsed.chapters.length : episode.chapters.length;
  const seek = hasAudio ? onSeek : undefined;

  return (
    <section className="card panel" aria-labelledby="episode-chapters-heading">
      <div className="panel-header">
        <div>
          <h2 className="panel-title" id="episode-chapters-heading">
            Chapters
          </h2>
          <p className="section-hint">
            {chapterCount} chapter{chapterCount === 1 ? "" : "s"} · Podcasting 2.0 JSON
          </p>
        </div>
        <a
          className="btn btn-secondary btn-sm"
          href={`${API_BASE}/episodes/${encodeURIComponent(episode.id)}/chapters`}
          download={`episode-${episode.number}-chapters.json`}
        >
          Download chapters JSON
        </a>
      </div>

      <div className="stack">
        <FileDrop
          id="episode-chapters-file"
          label="Choose a chapters .json file"
          hint="Drag and drop a Podcasting 2.0 chapters document here, or choose one."
          accept=".json,application/json"
          onFile={(file) => void handleFile(file)}
        />

        <TextAreaField
          id="episode-chapters-json"
          label="Chapters JSON"
          value={text}
          onChange={(event) => updateText(event.target.value)}
          className="textarea-code"
          rows={16}
          spellCheck={false}
          placeholder={
            '{\n  "version": "1.2.0",\n  "chapters": [\n    { "startTime": 0, "title": "Intro" }\n  ]\n}'
          }
          hint="The first chapter must start at 0 seconds; start times must be unique."
        />

        <div className="stack" aria-live="polite">
          {parsed && !parsed.ok ? (
            <ErrorBanner
              title="Fix the chapters document"
              message={`${parsed.errors.length} validation issue${parsed.errors.length === 1 ? "" : "s"} found.`}
              details={parsed.errors}
            />
          ) : null}
          {parsed?.ok ? (
            <InfoBanner
              title="The chapters document is valid"
              message={`${parsed.chapters.length} chapter${parsed.chapters.length === 1 ? "" : "s"}, starting at ${formatTimecode(parsed.chapters[0]?.start_seconds ?? 0)}.`}
            />
          ) : null}
          {saved ? (
            <div className="banner banner-success" role="status">
              <span className="banner-icon" aria-hidden="true">
                ✓
              </span>
              <span>Chapters saved.</span>
            </div>
          ) : null}
          {saveError ? <ErrorBanner message={saveError} /> : null}
        </div>

        <div className="row">
          <Button
            variant="primary"
            onClick={() => void handleSave()}
            disabled={!parsed?.ok || setChapters.isPending}
          >
            {setChapters.isPending ? "Saving…" : "Save chapters"}
          </Button>
          {sourceName ? <span className="muted text-small">Loaded from {sourceName}</span> : null}
        </div>

        {parsed?.ok ? (
          <div className="stack">
            <h3 className="section-title">Preview</h3>
            <ChapterTable chapters={parsed.chapters} onSeek={seek} caption="Chapter preview" />
            {hasAudio ? (
              <p className="muted text-small">Select a chapter to jump to it in the player.</p>
            ) : null}
          </div>
        ) : null}

        {!parsed && episode.chapters.length > 0 ? (
          <div className="stack">
            <h3 className="section-title">Saved chapters</h3>
            <ChapterTable chapters={episode.chapters} onSeek={seek} caption="Saved chapters" />
          </div>
        ) : null}

        {!parsed && episode.chapters.length === 0 ? (
          <p className="muted">
            No chapters saved yet. Upload or paste a chapters document to add some.
          </p>
        ) : null}
      </div>
    </section>
  );
}

function PublishPanel({ episode }: { episode: Episode }) {
  const publish = usePublishEpisode();
  const unpublish = useUnpublishEpisode();
  const [error, setError] = useState<string | null>(null);

  const hasAudio = episode.audio !== null;
  const hasChapters = episode.chapters.length > 0;
  const published = episode.status === "published";
  const busy = publish.isPending || unpublish.isPending;

  async function toggle() {
    setError(null);
    try {
      if (published) {
        await unpublish.mutateAsync(episode.id);
      } else {
        await publish.mutateAsync(episode.id);
      }
    } catch (toggleError) {
      setError(errorMessage(toggleError));
    }
  }

  return (
    <section className="card panel" aria-labelledby="episode-publish-heading">
      <div className="panel-header">
        <h2 className="panel-title" id="episode-publish-heading">
          Publishing
        </h2>
        <StatusBadge status={episode.status} />
      </div>

      <ul className="checklist">
        <li>
          <span
            className={hasAudio ? "check-icon check-icon-ok" : "check-icon check-icon-missing"}
            aria-hidden="true"
          >
            {hasAudio ? "✓" : "–"}
          </span>
          <span>Audio {hasAudio ? "uploaded" : "missing"}</span>
        </li>
        <li>
          <span
            className={hasChapters ? "check-icon check-icon-ok" : "check-icon check-icon-missing"}
            aria-hidden="true"
          >
            {hasChapters ? "✓" : "–"}
          </span>
          <span>
            {episode.chapters.length} chapter{episode.chapters.length === 1 ? "" : "s"}
            {hasChapters ? "" : " (at least one required)"}
          </span>
        </li>
      </ul>

      {episode.published_at ? (
        <p className="muted text-small mt-3">Published {formatDateTime(episode.published_at)}</p>
      ) : null}

      <div className="stack mt-4">
        {error ? <ErrorBanner message={error} /> : null}
        {published ? (
          <Button variant="secondary" onClick={() => void toggle()} disabled={busy}>
            {busy ? "Working…" : "Unpublish episode"}
          </Button>
        ) : (
          <Button
            variant="primary"
            onClick={() => void toggle()}
            disabled={busy || !hasAudio || !hasChapters}
          >
            {busy ? "Working…" : "Publish episode"}
          </Button>
        )}
        {!published && (!hasAudio || !hasChapters) ? (
          <p className="field-hint">
            An episode needs audio and at least one chapter before it can be published.
          </p>
        ) : null}
      </div>
    </section>
  );
}
