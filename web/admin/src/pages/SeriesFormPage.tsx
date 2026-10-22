import type { FormEvent } from "react";
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { errorMessage, isVersionConflict } from "../api/client";
import type { Series } from "../api/types";
import { Button } from "../components/Button";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { CoverTile } from "../components/CoverTile";
import { ErrorBanner } from "../components/ErrorBanner";
import { Field, TextAreaField, TextField } from "../components/Field";
import { FileDrop } from "../components/FileDrop";
import { LoadingBlock } from "../components/LoadingBlock";
import {
  useCreateSeries,
  useDeleteSeries,
  useRemoveCover,
  useSeries,
  useUpdateSeries,
  useUploadCover,
} from "../hooks/series";
import { isCoverImage } from "../lib/media";
import { deriveSlug, isValidSlug } from "../lib/slug";
import { withCacheBust } from "../lib/urls";

const DEFAULT_ACCENT = "#7A9CFF";
const MAX_IMAGE_BYTES = 10 * 1024 * 1024;
const HEX_COLOR = /^#[0-9A-Fa-f]{6}$/;

interface FormValues {
  title: string;
  slug: string;
  author: string;
  description: string;
  accentColor: string;
}

interface FormErrors {
  title?: string;
  slug?: string;
  author?: string;
  description?: string;
  accentColor?: string;
}

function validate(values: FormValues): FormErrors {
  const errors: FormErrors = {};
  const title = values.title.trim();
  if (title.length === 0) {
    errors.title = "A title is required.";
  } else if (title.length > 200) {
    errors.title = "The title must be at most 200 characters.";
  }

  const slug = values.slug.trim();
  if (slug.length > 0 && !isValidSlug(slug)) {
    errors.slug = "Use lowercase letters, digits and single dashes (maximum 120 characters).";
  }

  if (values.author.trim().length > 200) {
    errors.author = "The author must be at most 200 characters.";
  }
  if (values.description.length > 5000) {
    errors.description = "The description must be at most 5000 characters.";
  }
  if (!HEX_COLOR.test(values.accentColor)) {
    errors.accentColor = "Use a six-digit hex color such as #7A9CFF.";
  }
  return errors;
}

export function SeriesFormPage() {
  const { id } = useParams<{ id: string }>();
  const seriesQuery = useSeries(id);

  if (id) {
    if (seriesQuery.isPending) {
      return (
        <div className="page">
          <LoadingBlock label="Loading series…" />
        </div>
      );
    }
    if (seriesQuery.isError || !seriesQuery.data) {
      return (
        <div className="page">
          <ErrorBanner
            title="Could not load this series"
            message={
              seriesQuery.isError ? errorMessage(seriesQuery.error) : "The series was not found."
            }
          />
          <p className="mt-4">
            <Link to="/">Back to series</Link>
          </p>
        </div>
      );
    }
    return <SeriesForm key={seriesQuery.data.id} series={seriesQuery.data} />;
  }

  return <SeriesForm key="new" />;
}

function SeriesForm({ series }: { series?: Series }) {
  const navigate = useNavigate();
  const createSeries = useCreateSeries();
  const updateSeries = useUpdateSeries();
  const deleteSeries = useDeleteSeries();
  const uploadCover = useUploadCover();
  const removeCover = useRemoveCover();

  const [title, setTitle] = useState(series?.title ?? "");
  const [slug, setSlug] = useState(series?.slug ?? "");
  const [slugTouched, setSlugTouched] = useState(Boolean(series));
  const [author, setAuthor] = useState(series?.author ?? "");
  const [description, setDescription] = useState(series?.description ?? "");
  const [accentColor, setAccentColor] = useState(series?.accent_color ?? DEFAULT_ACCENT);
  const [submitted, setSubmitted] = useState(false);
  const [conflict, setConflict] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [coverError, setCoverError] = useState<string | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const values: FormValues = { title, slug, author, description, accentColor };
  const errors = validate(values);
  const invalid = Object.keys(errors).length > 0;
  const visibleError = (key: keyof FormErrors) => (submitted ? errors[key] : undefined);

  const saving = createSeries.isPending || updateSeries.isPending;

  const colorInputValue = HEX_COLOR.test(accentColor) ? accentColor : DEFAULT_ACCENT;

  function handleTitleChange(value: string) {
    setTitle(value);
    if (!slugTouched) setSlug(deriveSlug(value));
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitted(true);
    setConflict(false);
    setFormError(null);
    if (invalid) return;

    const input = {
      title: title.trim(),
      author: author.trim(),
      description,
      accent_color: accentColor.toUpperCase(),
    };

    try {
      if (series) {
        const updated = await updateSeries.mutateAsync({
          id: series.id,
          input: {
            ...input,
            slug: slug.trim() || series.slug,
            version: series.version,
          },
        });
        navigate(`/series/${updated.id}`);
      } else {
        const created = await createSeries.mutateAsync({
          ...input,
          ...(slug.trim() ? { slug: slug.trim() } : {}),
        });
        navigate(`/series/${created.id}`);
      }
    } catch (error) {
      if (isVersionConflict(error)) {
        setConflict(true);
      } else {
        setFormError(errorMessage(error));
      }
    }
  }

  async function handleCoverFile(file: File) {
    if (!series) return;
    setCoverError(null);
    if (!isCoverImage(file)) {
      setCoverError("Choose a JPEG, PNG or WebP image.");
      return;
    }
    if (file.size > MAX_IMAGE_BYTES) {
      setCoverError("The cover image must be 10 MB or smaller.");
      return;
    }
    try {
      await uploadCover.mutateAsync({ id: series.id, file });
    } catch (error) {
      setCoverError(errorMessage(error));
    }
  }

  async function handleRemoveCover() {
    if (!series) return;
    setCoverError(null);
    try {
      await removeCover.mutateAsync(series.id);
    } catch (error) {
      setCoverError(errorMessage(error));
    }
  }

  async function handleDelete() {
    if (!series) return;
    setDeleteError(null);
    try {
      await deleteSeries.mutateAsync(series.id);
      navigate("/");
    } catch (error) {
      setDeleteError(errorMessage(error));
    }
  }

  return (
    <div className="page">
      <nav className="breadcrumbs" aria-label="Breadcrumb">
        <Link to="/">Series</Link>
        <span aria-hidden="true">/</span>
        {series ? <Link to={`/series/${series.id}`}>{series.title}</Link> : <span>New series</span>}
        <span aria-hidden="true">/</span>
        <span aria-current="page">Edit</span>
      </nav>

      <header className="page-header">
        <div>
          <h1 className="page-title">{series ? "Edit series" : "New series"}</h1>
          <p className="page-subtitle">
            {series
              ? "Update the details and artwork of this series."
              : "Create a series, then add episodes and chapters."}
          </p>
        </div>
      </header>

      <form className="stack" onSubmit={handleSubmit} noValidate>
        {conflict ? (
          <ErrorBanner
            title="This series changed elsewhere"
            message="Someone or another tab updated this series. Reload the page to get the latest version before saving again."
          />
        ) : null}
        {conflict ? (
          <div>
            <Button variant="secondary" onClick={() => window.location.reload()}>
              Reload the page
            </Button>
          </div>
        ) : null}
        {formError ? <ErrorBanner message={formError} /> : null}

        <section className="card panel" aria-labelledby="series-details-heading">
          <div className="panel-header">
            <h2 className="panel-title" id="series-details-heading">
              Details
            </h2>
            {series ? <span className="muted text-small">Version {series.version}</span> : null}
          </div>

          <div className="stack">
            <TextField
              id="series-title"
              label="Title"
              value={title}
              onChange={(event) => handleTitleChange(event.target.value)}
              required
              maxLength={200}
              error={visibleError("title")}
              hint="Shown in the admin and used to derive the slug."
            />
            <TextField
              id="series-slug"
              label="Slug"
              value={slug}
              onChange={(event) => {
                setSlugTouched(true);
                setSlug(event.target.value);
              }}
              maxLength={120}
              error={visibleError("slug")}
              hint={
                series
                  ? "Public identifier of the series. Lowercase letters, digits and dashes."
                  : "Derived from the title until you edit it. Leave empty to let the server derive it."
              }
              autoComplete="off"
              spellCheck={false}
            />
            <TextField
              id="series-author"
              label="Author"
              value={author}
              onChange={(event) => setAuthor(event.target.value)}
              maxLength={200}
              error={visibleError("author")}
            />
            <TextAreaField
              id="series-description"
              label="Description"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              maxLength={5000}
              error={visibleError("description")}
              hint={`${description.length}/5000 characters`}
            />

            <Field
              id="series-accent-color"
              label="Accent color"
              hint="Used for the series tile when no cover is set."
              error={visibleError("accentColor")}
            >
              <div className="color-row">
                <input
                  className="color-swatch"
                  type="color"
                  aria-label="Pick the accent color"
                  value={colorInputValue}
                  onChange={(event) => setAccentColor(event.target.value)}
                />
                <input
                  id="series-accent-color"
                  className="input input-color-text"
                  type="text"
                  value={accentColor}
                  aria-invalid={visibleError("accentColor") ? true : undefined}
                  aria-describedby={
                    [visibleError("accentColor") ? "series-accent-color-error" : ""]
                      .filter(Boolean)
                      .join(" ") || undefined
                  }
                  onChange={(event) => setAccentColor(event.target.value)}
                  spellCheck={false}
                />
              </div>
            </Field>
          </div>
        </section>

        {series ? (
          <section className="card panel" aria-labelledby="series-cover-heading">
            <div className="panel-header">
              <h2 className="panel-title" id="series-cover-heading">
                Cover
              </h2>
            </div>
            <div className="detail-header">
              <CoverTile
                title={title || series.title}
                accentColor={colorInputValue}
                src={withCacheBust(series.cover_url, series.version)}
                size="lg"
              />
              <div className="stack">
                {coverError ? <ErrorBanner message={coverError} /> : null}
                <FileDrop
                  id="series-cover-file"
                  label={uploadCover.isPending ? "Uploading…" : "Upload cover image"}
                  hint="Drag and drop a JPEG, PNG or WebP image (10 MB maximum)."
                  accept="image/jpeg,image/png,image/webp"
                  disabled={uploadCover.isPending}
                  onFile={(file) => void handleCoverFile(file)}
                />
                {series.cover_url ? (
                  <div>
                    <Button
                      variant="ghost"
                      onClick={() => void handleRemoveCover()}
                      disabled={removeCover.isPending}
                    >
                      {removeCover.isPending ? "Removing…" : "Remove cover"}
                    </Button>
                  </div>
                ) : (
                  <p className="muted text-small">
                    No cover uploaded; the accent tile is shown instead.
                  </p>
                )}
              </div>
            </div>
          </section>
        ) : null}

        <div className="row">
          <Button type="submit" variant="primary" disabled={saving}>
            {saving ? "Saving…" : series ? "Save changes" : "Create series"}
          </Button>
          <Link className="btn btn-secondary" to={series ? `/series/${series.id}` : "/"}>
            Cancel
          </Link>
        </div>
      </form>

      {series ? (
        <section className="section card panel" aria-labelledby="series-danger-heading">
          <div className="panel-header">
            <h2 className="panel-title" id="series-danger-heading">
              Delete series
            </h2>
          </div>
          <p className="muted">
            Deleting a series removes it and its cover. This cannot be undone.
          </p>
          <div className="row mt-4">
            <Button
              variant="danger"
              onClick={() => {
                setDeleteError(null);
                setDeleteOpen(true);
              }}
              disabled={series.episode_count > 0}
            >
              Delete series
            </Button>
          </div>
          {series.episode_count > 0 ? (
            <p className="field-hint mt-2">
              This series still has {series.episode_count} episode
              {series.episode_count === 1 ? "" : "s"}. Delete the episodes before deleting the
              series.
            </p>
          ) : null}
        </section>
      ) : null}

      {series ? (
        <ConfirmDialog
          open={deleteOpen}
          title={`Delete “${series.title}”?`}
          description="The series and its cover will be permanently deleted. This action cannot be undone."
          confirmLabel="Delete series"
          busy={deleteSeries.isPending}
          error={deleteError}
          onCancel={() => setDeleteOpen(false)}
          onConfirm={() => void handleDelete()}
        />
      ) : null}
    </div>
  );
}
