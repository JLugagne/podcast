import type { FormEvent } from "react";
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { errorMessage } from "../api/client";
import { Button } from "../components/Button";
import { ErrorBanner } from "../components/ErrorBanner";
import { TextAreaField, TextField } from "../components/Field";
import { LoadingBlock } from "../components/LoadingBlock";
import { useCreateEpisode } from "../hooks/episodes";
import { useSeries } from "../hooks/series";

export function NewEpisodePage() {
  const { id: seriesId = "" } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const seriesQuery = useSeries(seriesId);
  const createEpisode = useCreateEpisode(seriesId);

  const [number, setNumber] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

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

  const series = seriesQuery.data;
  const parsedNumber = number.trim() ? Number.parseInt(number, 10) : undefined;
  const numberError =
    parsedNumber !== undefined && (!Number.isInteger(parsedNumber) || parsedNumber < 1)
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
    setFormError(null);
    if (invalid) return;

    try {
      const episode = await createEpisode.mutateAsync({
        ...(parsedNumber !== undefined ? { number: parsedNumber } : {}),
        title: title.trim(),
        description,
      });
      navigate(`/episodes/${episode.id}`);
    } catch (error) {
      setFormError(errorMessage(error));
    }
  }

  return (
    <div className="page">
      <nav className="breadcrumbs" aria-label="Breadcrumb">
        <Link to="/">Series</Link>
        <span aria-hidden="true">/</span>
        <Link to={`/series/${series.id}`}>{series.title}</Link>
        <span aria-hidden="true">/</span>
        <span aria-current="page">New episode</span>
      </nav>

      <header className="page-header">
        <div>
          <h1 className="page-title">New episode</h1>
          <p className="page-subtitle">Create the episode, then upload its audio and chapters.</p>
        </div>
      </header>

      <form className="stack" onSubmit={handleSubmit} noValidate>
        {formError ? <ErrorBanner message={formError} /> : null}

        <section className="card panel" aria-labelledby="new-episode-heading">
          <div className="panel-header">
            <h2 className="panel-title" id="new-episode-heading">
              Details
            </h2>
          </div>
          <div className="stack">
            <TextField
              id="new-episode-number"
              label="Number"
              type="number"
              min={1}
              step={1}
              value={number}
              onChange={(event) => setNumber(event.target.value)}
              placeholder="Next available number"
              hint="Leave empty to use the next available number in the series."
              error={submitted ? numberError : undefined}
            />
            <TextField
              id="new-episode-title"
              label="Title"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              required
              maxLength={200}
              error={submitted ? titleError : undefined}
            />
            <TextAreaField
              id="new-episode-description"
              label="Description"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              maxLength={10000}
              hint={`${description.length}/10000 characters`}
              error={submitted ? descriptionError : undefined}
            />
          </div>
        </section>

        <div className="row">
          <Button type="submit" variant="primary" disabled={createEpisode.isPending}>
            {createEpisode.isPending ? "Creating…" : "Create episode"}
          </Button>
          <Link className="btn btn-secondary" to={`/series/${series.id}`}>
            Cancel
          </Link>
        </div>
      </form>
    </div>
  );
}
