import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { AppShell } from "../App";
import { PlaybackProvider } from "../playback/PlaybackProvider";
import { FakeMediaEngine } from "./fakeEngine";

/** Mounts the real application routes with an isolated query cache and a fake engine. */
export function renderApp(initialPath = "/") {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, staleTime: 0, gcTime: 0 },
      mutations: { retry: false },
    },
  });
  const engine = new FakeMediaEngine();
  const user = userEvent.setup({ delay: null });
  const result = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialPath]}>
        <PlaybackProvider engine={engine}>
          <AppShell />
        </PlaybackProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { ...result, user, queryClient, engine };
}
