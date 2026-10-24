import { useMutation, useQueryClient } from "@tanstack/react-query";
import { addSubscription, refreshSubscription, removeSubscription } from "../lib/library";
import { queryKeys } from "./keys";

function useInvalidateCatalog() {
  const queryClient = useQueryClient();
  return () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.catalog });
  };
}

export function useAddPodcast() {
  const invalidate = useInvalidateCatalog();
  return useMutation({ mutationFn: addSubscription, onSuccess: invalidate });
}

export function useRefreshPodcast() {
  const invalidate = useInvalidateCatalog();
  return useMutation({ mutationFn: refreshSubscription, onSuccess: invalidate });
}

export function useRemovePodcast() {
  const invalidate = useInvalidateCatalog();
  return useMutation({
    mutationFn: (slug: string) => Promise.resolve(removeSubscription(slug)),
    onSuccess: invalidate,
  });
}
